package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// ResourceInfo tracks Podman resources for a managed project
type ResourceInfo struct {
	Containers []string `json:"containers"`
	Networks   []string `json:"networks"`
	Volumes    []string `json:"volumes"`
}

// ProjectState represents the state of a single managed project
type ProjectState struct {
	ProjectName string        `json:"project_name"`
	SourcePath  string        `json:"source_path"`
	Files       []string      `json:"files"`         // List of generated quadlet files
	Resources   *ResourceInfo `json:"resources,omitempty"` // Podman resources discovered via labels
}

// StateManager manages the persistence of project states in a JSON file
type StateManager struct {
	StateFilePath string
	Projects      map[string]ProjectState
}

// NewStateManager initializes a new state manager with a default path
func NewStateManager() (*StateManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(home, ".local", "share")
	}

	statePath := filepath.Join(dataDir, "comquad", "projects.json")

	sm := &StateManager{
		StateFilePath: statePath,
		Projects:      make(map[string]ProjectState),
	}

	// Load existing state if it exists
	if err := sm.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return sm, nil
}

func (sm *StateManager) load() error {
	data, err := os.ReadFile(sm.StateFilePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &sm.Projects)
}

func (sm *StateManager) Save() error {
	dir := filepath.Dir(sm.StateFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(sm.Projects, "", "  ")
	if err != nil {
		return err
	}

	// Write atomically: write to a temp file in the same directory, then rename.
	// This prevents a corrupted state file if the process is killed mid-write.
	tmp, err := os.CreateTemp(dir, ".projects-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	if err := os.Rename(tmpName, sm.StateFilePath); err != nil {
		os.Remove(tmpName)
		return err
	}

	return nil
}

func (sm *StateManager) RegisterProject(project ProjectState) error {
	// Preserve the existing Resources field if the caller didn't supply one.
	// This prevents up from silently clearing resource info written by regenerate.
	if project.Resources == nil {
		if existing, ok := sm.Projects[project.ProjectName]; ok {
			project.Resources = existing.Resources
		}
	}
	sm.Projects[project.ProjectName] = project
	return sm.Save()
}

func (sm *StateManager) UnregisterProject(projectName string) error {
	delete(sm.Projects, projectName)
	return sm.Save()
}

func (sm *StateManager) ListProjects() []ProjectState {
	projects := make([]ProjectState, 0, len(sm.Projects))
	for _, p := range sm.Projects {
		projects = append(projects, p)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].ProjectName < projects[j].ProjectName
	})
	return projects
}
