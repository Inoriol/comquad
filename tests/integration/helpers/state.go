//go:build integration

package helpers

import (
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
)

type ProjectState struct {
    ProjectName string   `json:"project_name"`
    SourcePath  string   `json:"source_path"`
    Files       []string `json:"files"`
    Resources   struct {
        Containers []string `json:"containers"`
        Networks   []string `json:"networks"`
        Volumes    []string `json:"volumes"`
    } `json:"resources"`
}

// ReadStateFile parses projects.json and returns all project entries.
// Returns an empty map if the state file does not exist yet (e.g. before first deploy).
func ReadStateFile(t *testing.T) map[string]ProjectState {
    t.Helper()

    stateDir := os.Getenv("XDG_DATA_HOME")
    if stateDir == "" {
        home, err := os.UserHomeDir()
        if err != nil {
            t.Fatalf("could not determine home dir: %v", err)
        }
        stateDir = filepath.Join(home, ".local", "share")
    }

    path := filepath.Join(stateDir, "comquad", "projects.json")
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return make(map[string]ProjectState)
        }
        t.Fatalf("failed to read state file at %s: %v", path, err)
    }

    var projects map[string]ProjectState
    if err := json.Unmarshal(data, &projects); err != nil {
        t.Fatalf("failed to parse state file: %v", err)
    }
    return projects
}

// AssertProjectRegistered fails if the project is not in projects.json.
func AssertProjectRegistered(t *testing.T, projectName string) ProjectState {
    t.Helper()
    projects := ReadStateFile(t)
    p, ok := projects[projectName]
    if !ok {
        t.Fatalf("project %q not found in state file", projectName)
    }
    return p
}

// AssertProjectGone fails if the project is still in projects.json.
func AssertProjectGone(t *testing.T, projectName string) {
    t.Helper()
    projects := ReadStateFile(t)
    if _, ok := projects[projectName]; ok {
        t.Fatalf("project %q still present in state file after down", projectName)
    }
}
