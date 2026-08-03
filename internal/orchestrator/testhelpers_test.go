package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"comquad/internal/deploy"
	"github.com/coreos/go-systemd/v22/dbus"
)

// ---------------------------------------------------------------------------
// mockStateStore — in-memory StateStore for tests
// ---------------------------------------------------------------------------

type mockStateStore struct {
	mu       sync.Mutex
	projects map[string]deploy.ProjectState
	filePath string
	saveErr  error // if set, Save/RegisterProject/UnregisterProject returns this
}

func newMockStateStore(projects map[string]deploy.ProjectState) *mockStateStore {
	if projects == nil {
		projects = make(map[string]deploy.ProjectState)
	}
	return &mockStateStore{
		projects: projects,
		filePath: "/fake/projects.json",
	}
}

func (m *mockStateStore) GetProject(name string) (deploy.ProjectState, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[name]
	return p, ok
}

func (m *mockStateStore) GetStateFilePath() string { return m.filePath }

func (m *mockStateStore) ListProjects() []deploy.ProjectState {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps := make([]deploy.ProjectState, 0, len(m.projects))
	for _, p := range m.projects {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].ProjectName < ps[j].ProjectName })
	return ps
}

func (m *mockStateStore) RegisterProject(p deploy.ProjectState) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[p.ProjectName] = p
	return nil
}

func (m *mockStateStore) UnregisterProject(name string) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.projects, name)
	return nil
}

func (m *mockStateStore) Save() error { return m.saveErr }

// ---------------------------------------------------------------------------
// mockSystemdClient — controllable SystemdClient for tests
// ---------------------------------------------------------------------------

type unitRecord struct {
	name        string
	activeState string
	subState    string
}

type mockSystemdClient struct {
	mu sync.Mutex

	// Recorded calls
	reloadCalls  [][]string // each element is the filePaths slice passed
	startedUnits []string
	stoppedUnits []string
	restarted    []string

	// Canned units returned by ListAllUnits / ListUnitsByNames
	units []unitRecord

	// Invocation IDs keyed by unit name
	invocationIDs map[string]string

	// Per-method error injection
	reloadErr  error
	startErr   map[string]error // keyed by unit name, falls back to startErrDefault
	startErrDefault error
	stopErr    map[string]error
	restartErr map[string]error
	listErr    error
	waitErr    error
}

func newMockSystemdClient() *mockSystemdClient {
	return &mockSystemdClient{
		startErr:      make(map[string]error),
		stopErr:       make(map[string]error),
		restartErr:    make(map[string]error),
		invocationIDs: make(map[string]string),
	}
}

func (m *mockSystemdClient) Close() error { return nil }

func (m *mockSystemdClient) ReloadDaemon(filePaths ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadCalls = append(m.reloadCalls, filePaths)
	return m.reloadErr
}

func (m *mockSystemdClient) WaitForUnit(_ string, _ time.Duration) error {
	return m.waitErr
}

func (m *mockSystemdClient) StartUnit(unitName string) error {
	if err, ok := m.startErr[unitName]; ok {
		return err
	}
	if m.startErrDefault != nil {
		return m.startErrDefault
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startedUnits = append(m.startedUnits, unitName)
	return nil
}

func (m *mockSystemdClient) StopUnit(unitName string) error {
	if err, ok := m.stopErr[unitName]; ok {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stoppedUnits = append(m.stoppedUnits, unitName)
	return nil
}

func (m *mockSystemdClient) RestartUnit(unitName string) error {
	if err, ok := m.restartErr[unitName]; ok {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restarted = append(m.restarted, unitName)
	return nil
}

func (m *mockSystemdClient) ListUnitsByNames(names []string) ([]dbus.UnitStatus, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []dbus.UnitStatus
	for _, name := range names {
		for _, u := range m.units {
			if u.name == name {
				out = append(out, dbus.UnitStatus{
					Name:        u.name,
					ActiveState: u.activeState,
					SubState:    u.subState,
				})
			}
		}
	}
	return out, nil
}

func (m *mockSystemdClient) ListAllUnits() ([]dbus.UnitStatus, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []dbus.UnitStatus
	for _, u := range m.units {
		out = append(out, dbus.UnitStatus{
			Name:        u.name,
			ActiveState: u.activeState,
			SubState:    u.subState,
		})
	}
	return out, nil
}

func (m *mockSystemdClient) GetInvocationID(unitName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.invocationIDs[unitName], nil
}

// ---------------------------------------------------------------------------
// Orchestrator constructor for tests
// ---------------------------------------------------------------------------

// newTestOrchestrator creates an Orchestrator with injected fakes.
// cwd should be a temp directory created by the caller.
func newTestOrchestrator(projectName, cwd string, state *mockStateStore, sys *mockSystemdClient) *Orchestrator {
	return &Orchestrator{
		projectName: projectName,
		cwd:         cwd,
		newState: func() (deploy.StateStore, error) {
			return state, nil
		},
		newSystemd: func() (deploy.SystemdClient, error) {
			return sys, nil
		},
		newJournalCmd: func(name string, args ...string) *exec.Cmd {
			return exec.Command(name, args...)
		},
	}
}

// newTestOrchestratorWithStateErr returns an orchestrator whose newState factory
// always returns the given error.
func newTestOrchestratorWithStateErr(projectName, cwd string, err error) *Orchestrator {
	return &Orchestrator{
		projectName: projectName,
		cwd:         cwd,
		newState: func() (deploy.StateStore, error) {
			return nil, err
		},
		newSystemd: func() (deploy.SystemdClient, error) {
			return newMockSystemdClient(), nil
		},
		newJournalCmd: func(name string, args ...string) *exec.Cmd {
			return exec.Command(name, args...)
		},
	}
}

// newTestOrchestratorWithSystemdErr returns an orchestrator whose newSystemd
// factory always returns the given error.
func newTestOrchestratorWithSystemdErr(projectName, cwd string, state *mockStateStore, err error) *Orchestrator {
	return &Orchestrator{
		projectName: projectName,
		cwd:         cwd,
		newState: func() (deploy.StateStore, error) {
			return state, nil
		},
		newSystemd: func() (deploy.SystemdClient, error) {
			return nil, err
		},
		newJournalCmd: func(name string, args ...string) *exec.Cmd {
			return exec.Command(name, args...)
		},
	}
}

// ---------------------------------------------------------------------------
// Filesystem helpers for tests
// ---------------------------------------------------------------------------

// writeContainerFile writes a minimal quadlet container file to dir and
// returns its absolute path.
func writeContainerFile(t interface{ Helper(); Fatal(...interface{}); TempDir() string }, dir, name string) string {
	path := filepath.Join(dir, name)
	content := "[Container]\nImage=docker.io/library/nginx\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(fmt.Sprintf("writeContainerFile: %v", err))
	}
	return path
}

// makeProjectState returns a ProjectState populated with files all living in dir.
func makeProjectState(name, srcPath string, files []string) deploy.ProjectState {
	return deploy.ProjectState{
		ProjectName: name,
		SourcePath:  srcPath,
		Files:       files,
	}
}
