package orchestrator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/reconcile"
)

func makeMinimalCompose(t *testing.T, dir string) {
	t.Helper()
	content := `services:
  web:
    image: nginx
`
	writeFile(t, filepath.Join(dir, "compose.yaml"), content)
}

func TestUp_NoComposeFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())
	o.cwd = dir

	err := o.Up("missing", false, false, true)
	if err == nil || !strings.Contains(err.Error(), "no compose file found") {
		t.Errorf("expected 'no compose file found' error, got %v", err)
	}
}

func TestUp_InvalidYamlReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "compose.yaml"), "{invalid")
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())
	o.cwd = dir

	err := o.Up("missing", false, false, true)
	if err == nil {
		t.Error("expected error for invalid YAML")
	} else if !strings.Contains(err.Error(), "transpile") &&
		!strings.Contains(err.Error(), "unmarshal") &&
		!strings.Contains(err.Error(), "yaml") {
		t.Errorf("expected transpile/yaml-related error, got: %v", err)
	}
}

func TestUp_StateRegistrationError(t *testing.T) {
	dir := t.TempDir()
	makeMinimalCompose(t, dir)

	o := newTestOrchestratorWithStateErr("myapp", dir, errors.New("cannot write state"))
	o.cwd = dir

	err := o.Up("missing", false, false, true)
	if err == nil || !strings.Contains(err.Error(), "cannot write state") {
		t.Errorf("expected 'cannot write state' error, got %v", err)
	}
}

func TestUp_InvalidPullStrategyReturnsError(t *testing.T) {
	dir := t.TempDir()
	makeMinimalCompose(t, dir)

	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())
	o.cwd = dir

	err := o.Up("badstrategy", false, false, true)
	if err == nil || !strings.Contains(err.Error(), "invalid pull strategy") {
		t.Errorf("expected 'unknown pull strategy' error, got %v", err)
	}
}

func TestRegisterState_PersistsToStateStore(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())
	o.cwd = dir

	files := []string{
		filepath.Join(dir, "cq-myapp-web.container"),
	}

	if err := o.registerState(files); err != nil {
		t.Fatalf("registerState failed: %v", err)
	}

	p, exists := state.projects["myapp"]
	if !exists {
		t.Fatal("expected project to be registered")
	}
	if p.ProjectName != "myapp" {
		t.Errorf("expected ProjectName 'myapp', got %q", p.ProjectName)
	}
	if p.SourcePath != dir {
		t.Errorf("expected SourcePath %q, got %q", dir, p.SourcePath)
	}
	if len(p.Files) != 1 || p.Files[0] != files[0] {
		t.Errorf("expected files %v, got %v", files, p.Files)
	}
}

func TestRegisterState_StateError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(nil)
	state.saveErr = errors.New("disk full")
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())
	o.cwd = dir

	err := o.registerState([]string{})
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Errorf("expected disk full error, got %v", err)
	}
}

func TestCollectProjectFiles_ReturnsProjectFiles(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{
		"cq-myapp-web.container",
		"cq-myapp-default.network",
		"cq-other-web.container",
		"cq-myapp2-web.container",
	} {
		writeFile(t, filepath.Join(dir, name), "")
	}

	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	files, err := o.collectProjectFiles(dir)
	if err != nil {
		t.Fatalf("collectProjectFiles: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files for myapp, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if !strings.Contains(f, "cq-myapp-") {
			t.Errorf("unexpected file in results: %q", f)
		}
	}
}

func TestCollectProjectFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	files, err := o.collectProjectFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %v", files)
	}
}

func TestCollectProjectFiles_NonExistentDirReturnsError(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	_, err := o.collectProjectFiles("/nonexistent/path/xyz")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestRollbackDeploy_RestoresPreviousState(t *testing.T) {
	dir := t.TempDir()
	targetDir := t.TempDir()
	baselineDir := t.TempDir()

	oldContent := "[Container]\nImage=docker.io/library/nginx:alpine\n"
	targetWeb := filepath.Join(targetDir, "cq-myapp-web.container")
	baseWeb := filepath.Join(baselineDir, "cq-myapp-web.container")
	writeFile(t, targetWeb, oldContent)
	writeFile(t, baseWeb, oldContent)

	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{{Key: "Image", Values: []string{"docker.io/library/nginx:alpine-v2"}}}},
			},
		},
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-db",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{{Key: "Image", Values: []string{"docker.io/library/postgres:15"}}}},
			},
		},
	}

	plan, err := reconcile.Compute(targetDir, baselineDir, "cq-myapp-", units)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reconcile.Apply(targetDir, baselineDir, plan); err != nil {
		t.Fatal(err)
	}

	// Simulate the partially-applied state of a failed deploy.
	data, err := os.ReadFile(targetWeb)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "nginx:alpine-v2") {
		t.Fatalf("expected new content before rollback, got: %s", data)
	}
	dbTarget := filepath.Join(targetDir, "cq-myapp-db.container")
	if _, err := os.Stat(dbTarget); err != nil {
		t.Fatalf("expected db container created before rollback: %v", err)
	}

	priorState := makeProjectState("myapp", dir, []string{targetWeb})
	state := newMockStateStore(map[string]deploy.ProjectState{"myapp": priorState})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	o.rollbackDeploy(plan, priorState, true)

	data, err = os.ReadFile(targetWeb)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != oldContent {
		t.Errorf("expected web file restored to previous content, got: %q", data)
	}

	if _, err := os.Stat(dbTarget); !os.IsNotExist(err) {
		t.Error("expected db container to be removed after rollback")
	}

	data, err = os.ReadFile(baseWeb)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != oldContent {
		t.Errorf("expected web baseline restored, got: %q", data)
	}
	if _, err := os.Stat(filepath.Join(baselineDir, "cq-myapp-db.container")); !os.IsNotExist(err) {
		t.Error("expected db baseline to be removed after rollback")
	}
}
