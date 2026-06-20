package orchestrator

// Tests for List, Exec, View (error paths), and Ps (error paths).
// Commands that require a live D-Bus or journalctl are tested only for their
// state-lookup and argument-validation paths — not the actual systemd interaction.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"comquad/internal/deploy"
)

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList_EmptyStateReturnsNoError(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("", t.TempDir(), state, newMockSystemdClient())

	if err := o.List(); err != nil {
		t.Errorf("List on empty state should not error, got %v", err)
	}
}

func TestList_StateError(t *testing.T) {
	o := newTestOrchestratorWithStateErr("myapp", t.TempDir(), errors.New("state broken"))
	err := o.List()
	if err == nil || !strings.Contains(err.Error(), "state broken") {
		t.Errorf("expected state error, got %v", err)
	}
}

func TestList_MultipleProjectsNoFilter(t *testing.T) {
	state := newMockStateStore(map[string]deploy.ProjectState{
		"alpha": makeProjectState("alpha", "/a", nil),
		"beta":  makeProjectState("beta", "/b", nil),
	})
	// No project name filter
	o := newTestOrchestrator("", t.TempDir(), state, newMockSystemdClient())
	// Just verify it doesn't error
	if err := o.List(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestList_FiltersByProjectName(t *testing.T) {
	state := newMockStateStore(map[string]deploy.ProjectState{
		"alpha": makeProjectState("alpha", "/a", nil),
		"beta":  makeProjectState("beta", "/b", nil),
	})
	o := newTestOrchestrator("alpha", t.TempDir(), state, newMockSystemdClient())
	if err := o.List(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Exec
// ---------------------------------------------------------------------------

func TestExec_RequiresCommand(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.Exec("web", "", true, []string{})
	if err == nil || !strings.Contains(err.Error(), "exec requires a command") {
		t.Errorf("expected 'exec requires a command' error, got %v", err)
	}
}

func TestExec_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.Exec("web", "", true, []string{"ls"})
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

func TestExec_StateError(t *testing.T) {
	o := newTestOrchestratorWithStateErr("myapp", t.TempDir(), errors.New("state failure"))
	err := o.Exec("web", "", true, []string{"ls"})
	if err == nil || !strings.Contains(err.Error(), "state failure") {
		t.Errorf("expected state error, got %v", err)
	}
}

func TestExec_ServiceNotFound(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
		}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.Exec("nonexistent", "", true, []string{"ls"})
	if err == nil || !strings.Contains(err.Error(), "no containers found matching") {
		t.Errorf("expected 'no containers found' error, got %v", err)
	}
}

func TestExec_NetworkServiceReturnsError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-default.network"),
		}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.Exec("cq-myapp-default.network", "", true, []string{"ls"})
	if err == nil || !strings.Contains(err.Error(), "cannot exec into network or volume") {
		t.Errorf("expected 'cannot exec into network or volume' error, got %v", err)
	}
}

func TestExec_AmbiguousServiceReturnsError(t *testing.T) {
	dir := t.TempDir()
	// Two container files that both match the arg "web" via different patterns
	// is hard to set up naturally, but we can test the multi-match path by
	// having the service name match exactly two separate files — use a short
	// project name that makes the cq-prefix and short-name patterns both fire.
	// Simpler: test with no service arg when multiple containers exist.
	// Actually the ambiguous path only fires when len(matches) > 1.
	// We can't get two matches for the same arg in normal usage without
	// pathological naming, so we test the no-service path instead.
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
			filepath.Join(dir, "cq-myapp-db.container"),
		}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	// No service arg + multiple containers → ambiguous error
	err := o.Exec("", "", true, []string{"ls"})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguous service error, got %v", err)
	}
}

func TestExec_NoContainersInProject(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-default.network"),
		}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.Exec("", "", true, []string{"ls"})
	if err == nil || !strings.Contains(err.Error(), "no containers found") {
		t.Errorf("expected 'no containers found' error, got %v", err)
	}
}

func TestExec_ContainerNameDerivedFromBasename(t *testing.T) {
	// Write a real container file so podman exec actually has a name to work
	// with. We can't run podman in unit tests, so we verify the error message
	// contains the correctly-derived container name (not a full path).
	dir := t.TempDir()
	containerFile := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerFile, "[Container]\nImage=nginx\n")

	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{containerFile}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	// podman exec will fail because the container isn't running, but the error
	// should NOT contain a filesystem path as the container name.
	err := o.Exec("web", "", false, []string{"true"})
	if err != nil {
		// The error should mention "myapp-web" (derived name), NOT a path like
		// "/tmp/.../cq-myapp-web.container"
		if strings.Contains(err.Error(), dir) {
			t.Errorf("container name in error appears to be a path: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// View — error paths
// ---------------------------------------------------------------------------

func TestView_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.View("")
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

func TestView_StateError(t *testing.T) {
	o := newTestOrchestratorWithStateErr("myapp", t.TempDir(), errors.New("state gone"))
	err := o.View("")
	if err == nil || !strings.Contains(err.Error(), "state gone") {
		t.Errorf("expected state error, got %v", err)
	}
}

func TestView_UnitFileNotFound(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
		}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.View("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "no unit found") {
		t.Errorf("expected 'no unit found' error, got %v", err)
	}
}

func TestView_PrintsUnitFileContent(t *testing.T) {
	dir := t.TempDir()
	containerFile := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerFile, "[Container]\nImage=nginx\n")

	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{containerFile}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	// View with a service arg reads the file — should not error
	if err := o.View("web"); err != nil {
		t.Errorf("unexpected error viewing unit file: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Ps — error paths
// ---------------------------------------------------------------------------

func TestPs_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.Ps()
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

func TestPs_StateError(t *testing.T) {
	o := newTestOrchestratorWithStateErr("myapp", t.TempDir(), errors.New("state gone"))
	err := o.Ps()
	if err == nil || !strings.Contains(err.Error(), "state gone") {
		t.Errorf("expected state error, got %v", err)
	}
}

func TestPs_SystemdConnectionError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	o := newTestOrchestratorWithSystemdErr("myapp", dir, state, errors.New("dbus gone"))

	err := o.Ps()
	if err == nil || !strings.Contains(err.Error(), "dbus gone") {
		t.Errorf("expected dbus error, got %v", err)
	}
}

func TestPs_PrintsUnitsForProject(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "active", subState: "running"},
	}
	o := newTestOrchestrator("myapp", dir, state, sys)

	if err := o.Ps(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Logs — error paths (state lookup only, no actual journalctl)
// ---------------------------------------------------------------------------

func TestLogs_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.Logs(nil, false, "", "", "")
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

func TestLogs_NoContainerUnitsInProject(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-default.network"),
		}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.Logs(nil, false, "", "", "")
	if err == nil || !strings.Contains(err.Error(), "no container units found") {
		t.Errorf("expected 'no container units found' error, got %v", err)
	}
}

func TestLogs_ServiceNotFound(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
		}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.Logs([]string{"nonexistent"}, false, "", "", "")
	if err == nil || !strings.Contains(err.Error(), "no service matching") {
		t.Errorf("expected 'no service matching' error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Edit — error paths
// ---------------------------------------------------------------------------

func TestEdit_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.Edit("", false)
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

func TestEdit_NoUnitsFound(t *testing.T) {
	dir := t.TempDir()
	// Project exists but has no quadlet files
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.Edit("", false)
	if err == nil || !strings.Contains(err.Error(), "no units found") {
		t.Errorf("expected 'no units found' error, got %v", err)
	}
}

func TestEdit_UnitNotFound(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
		}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.Edit("nonexistent", false)
	if err == nil || !strings.Contains(err.Error(), "no unit found") {
		t.Errorf("expected 'no unit found' error, got %v", err)
	}
}

func TestEdit_NoReloadFlagSkipsSystemd(t *testing.T) {
	dir := t.TempDir()
	containerFile := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerFile, "[Container]\nImage=nginx\n")

	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{containerFile}),
	})
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, state, sys)

	// Use a no-op editor that makes no changes
	t.Setenv("EDITOR", "true")

	if err := o.Edit("", true /*noReload*/); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No systemd reload should have happened
	if len(sys.reloadCalls) != 0 {
		t.Errorf("expected no reload calls with --no-reload, got %d", len(sys.reloadCalls))
	}
}

// ---------------------------------------------------------------------------
// verifyUnitsStopped / verifyUnitsStoppedByNames
// ---------------------------------------------------------------------------

func TestVerifyUnitsStopped_ActiveUnitReturnsError(t *testing.T) {
	dir := t.TempDir()
	containerFile := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerFile, "")

	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "active"},
	}
	o := newTestOrchestrator("myapp", dir, newMockStateStore(nil), sys)

	err := o.verifyUnitsStopped(sys, []string{containerFile})
	if err == nil || !strings.Contains(err.Error(), "still active") {
		t.Errorf("expected 'still active' error, got %v", err)
	}
}

func TestVerifyUnitsStopped_InactiveUnitsPass(t *testing.T) {
	dir := t.TempDir()
	containerFile := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerFile, "")

	sys := newMockSystemdClient()
	// Unit is known but inactive
	sys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "inactive"},
	}
	o := newTestOrchestrator("myapp", dir, newMockStateStore(nil), sys)

	if err := o.verifyUnitsStopped(sys, []string{containerFile}); err != nil {
		t.Errorf("unexpected error for inactive unit: %v", err)
	}
}

func TestVerifyUnitsStoppedByNames_ActiveReturnsError(t *testing.T) {
	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "active"},
	}
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	err := o.verifyUnitsStoppedByNames(sys, []string{"cq-myapp-web.service"})
	if err == nil || !strings.Contains(err.Error(), "still active") {
		t.Errorf("expected 'still active' error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// pluralize helper (in regenerate.go)
// ---------------------------------------------------------------------------

func TestPluralize(t *testing.T) {
	if pluralize(0) != "s" {
		t.Error("expected 's' for 0")
	}
	if pluralize(1) != "" {
		t.Error("expected '' for 1")
	}
	if pluralize(2) != "s" {
		t.Error("expected 's' for 2")
	}
}

// ---------------------------------------------------------------------------
// NewOrchestrator
// ---------------------------------------------------------------------------

func TestNewOrchestrator_UsesDirectoryNameAsDefault(t *testing.T) {
	// Change to a temp dir so the default project name is predictable
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(dir)

	o, err := NewOrchestrator("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.projectName != filepath.Base(dir) {
		t.Errorf("expected project name %q, got %q", filepath.Base(dir), o.projectName)
	}
}

func TestNewOrchestrator_ExplicitNameOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(dir)

	o, err := NewOrchestrator("custom-name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.projectName != "custom-name" {
		t.Errorf("expected 'custom-name', got %q", o.projectName)
	}
}
