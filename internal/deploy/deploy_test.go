package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateManager_RegisterAndUnregister(t *testing.T) {
	sm, err := NewStateManager()
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}

	projects := sm.ListProjects()
	initialCount := len(projects)

	proj := ProjectState{
		ProjectName: "testproj",
		SourcePath:  "/tmp/test",
		Files:       []string{"/tmp/f1", "/tmp/f2"},
	}

	if err := sm.RegisterProject(proj); err != nil {
		t.Fatalf("RegisterProject failed: %v", err)
	}

	projects = sm.ListProjects()
	if len(projects) != initialCount+1 {
		t.Errorf("expected %d projects, got %d", initialCount+1, len(projects))
	}

	if _, exists := sm.Projects["testproj"]; !exists {
		t.Error("expected project to be in Projects map")
	}

	if err := sm.UnregisterProject("testproj"); err != nil {
		t.Fatalf("UnregisterProject failed: %v", err)
	}

	if _, exists := sm.Projects["testproj"]; exists {
		t.Error("expected project to be removed from Projects map")
	}
}

func TestStateManager_PersistsToDisk(t *testing.T) {
	sm, err := NewStateManager()
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}

	proj := ProjectState{
		ProjectName: "persist-test",
		SourcePath:  "/tmp/persist",
		Files:       []string{"/tmp/pfile"},
	}

	if err := sm.RegisterProject(proj); err != nil {
		t.Fatalf("RegisterProject failed: %v", err)
	}

	// Load a fresh StateManager to verify persistence
	sm2, err := NewStateManager()
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}

	if _, exists := sm2.Projects["persist-test"]; !exists {
		t.Error("expected persisted project to be loaded in fresh StateManager")
	}
}

func TestStateManager_ListProjects(t *testing.T) {
	sm, err := NewStateManager()
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}

	// Clear existing projects first
	for name := range sm.Projects {
		delete(sm.Projects, name)
	}
	sm.Save()

	projects := sm.ListProjects()
	if len(projects) != 0 {
		t.Errorf("expected 0 projects after clearing, got %d", len(projects))
	}

	sm.Projects["p1"] = ProjectState{ProjectName: "p1", SourcePath: "/p1", Files: []string{"/f1"}}
	sm.Projects["p2"] = ProjectState{ProjectName: "p2", SourcePath: "/p2", Files: []string{"/f2"}}

	projects = sm.ListProjects()
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestStateManager_DoesNotFailOnMissingStateFile(t *testing.T) {
	// Create a temp dir and set XDG_DATA_HOME to a non-existent location
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", filepath.Join(os.TempDir(), "comquad-test-nodir"))
	defer os.Setenv("XDG_DATA_HOME", oldXDG)

	sm, err := NewStateManager()
	if err != nil {
		t.Fatalf("NewStateManager should not fail on missing state file: %v", err)
	}

	if sm.Projects == nil {
		t.Error("expected Projects map to be initialized")
	}
}

func TestNewTargetDirResolver_Rootless(t *testing.T) {
	// We can't easily test as root in CI, so just verify the resolver works
	r := NewTargetDirResolver()
	path, err := r.GetSystemdPath()
	if err != nil {
		t.Fatalf("GetSystemdPath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestTargetDirResolver_ReturnsValidPath(t *testing.T) {
	r := NewTargetDirResolver()
	path, err := r.GetSystemdPath()
	if err != nil {
		t.Fatalf("GetSystemdPath failed: %v", err)
	}

	// Path should be absolute
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %q", path)
	}
}
