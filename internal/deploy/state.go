package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ProjectState represents the state of a single managed project
type ProjectState struct {
	ProjectName string   `json:"project_name"`
	SourcePath  string   `json:"source_path"`
	Files       []string `json:"files"` // List of generated quadlet files
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
	err := os.MkdirAll(filepath.Dir(sm.StateFilePath), 0755)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(sm.Projects, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sm.StateFilePath, data, 0644)
}

func (sm *StateManager) RegisterProject(project ProjectState) error {
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
	return projects
}
