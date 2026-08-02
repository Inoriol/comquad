package orchestrator

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"comquad/internal/deploy"
)

// ---------------------------------------------------------------------------
// resolveUnits
// ---------------------------------------------------------------------------

func TestResolveUnits_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	_, err := o.resolveUnits(nil)
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

func TestResolveUnits_StateError(t *testing.T) {
	o := newTestOrchestratorWithStateErr("myapp", t.TempDir(), errors.New("db failure"))

	_, err := o.resolveUnits(nil)
	if err == nil || !strings.Contains(err.Error(), "db failure") {
		t.Errorf("expected state error propagated, got %v", err)
	}
}

func TestResolveUnits_AllContainersWhenNoServicesGiven(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "cq-myapp-web.container"),
		filepath.Join(dir, "cq-myapp-db.container"),
		filepath.Join(dir, "cq-myapp-default.network"),
	}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	units, err := o.resolveUnits(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 2 {
		t.Errorf("expected 2 container units, got %d: %v", len(units), units)
	}
	for _, u := range units {
		if !strings.HasSuffix(u, ".service") {
			t.Errorf("expected .service suffix, got %q", u)
		}
	}
}

func TestResolveUnits_SpecificService(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "cq-myapp-web.container"),
		filepath.Join(dir, "cq-myapp-db.container"),
	}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	units, err := o.resolveUnits([]string{"web"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 1 {
		t.Errorf("expected 1 unit, got %d: %v", len(units), units)
	}
	if units[0] != "cq-myapp-web.service" {
		t.Errorf("expected cq-myapp-web.service, got %q", units[0])
	}
}

func TestResolveUnits_UnknownServiceReturnsError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
		}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	_, err := o.resolveUnits([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown service")
	}
}

func TestResolveUnits_DeduplicatesResults(t *testing.T) {
	dir := t.TempDir()
	files := []string{filepath.Join(dir, "cq-myapp-web.container")}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	// Pass "web" twice — should deduplicate to one unit
	units, err := o.resolveUnits([]string{"web", "web"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 1 {
		t.Errorf("expected 1 unit after dedup, got %d: %v", len(units), units)
	}
}

// ---------------------------------------------------------------------------
// Start
// ---------------------------------------------------------------------------

func TestStart_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.Start(nil, false)
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

func TestStart_CallsStartUnitForEachContainer(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "cq-myapp-web.container"),
		filepath.Join(dir, "cq-myapp-db.container"),
	}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, state, sys)

	if err := o.Start(nil, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sys.startedUnits) != 2 {
		t.Errorf("expected 2 started units, got %d: %v", len(sys.startedUnits), sys.startedUnits)
	}
}

func TestStart_SpecificService(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "cq-myapp-web.container"),
		filepath.Join(dir, "cq-myapp-db.container"),
	}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, state, sys)

	if err := o.Start([]string{"web"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sys.startedUnits) != 1 || sys.startedUnits[0] != "cq-myapp-web.service" {
		t.Errorf("expected only web started, got %v", sys.startedUnits)
	}
}

func TestStart_PropagatesStartError(t *testing.T) {
	dir := t.TempDir()
	files := []string{filepath.Join(dir, "cq-myapp-web.container")}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	sys := newMockSystemdClient()
	sys.startErrDefault = errors.New("unit failed")
	o := newTestOrchestrator("myapp", dir, state, sys)

	err := o.Start(nil, false)
	if err == nil || !strings.Contains(err.Error(), "unit failed") {
		t.Errorf("expected start error propagated, got %v", err)
	}
}

func TestStart_SystemdConnectionError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
		}),
	})
	o := newTestOrchestratorWithSystemdErr("myapp", dir, state, errors.New("dbus down"))

	err := o.Start(nil, false)
	if err == nil || !strings.Contains(err.Error(), "dbus down") {
		t.Errorf("expected dbus error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Stop
// ---------------------------------------------------------------------------

func TestStop_CallsStopUnitForEachContainer(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "cq-myapp-web.container"),
		filepath.Join(dir, "cq-myapp-db.container"),
	}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	sys := newMockSystemdClient()
	// Units are not active, so verifyUnitsStoppedByNames should pass
	o := newTestOrchestrator("myapp", dir, state, sys)

	if err := o.Stop(nil, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sys.stoppedUnits) != 2 {
		t.Errorf("expected 2 stopped units, got %d: %v", len(sys.stoppedUnits), sys.stoppedUnits)
	}
}

func TestStop_PropagatesStopError(t *testing.T) {
	dir := t.TempDir()
	files := []string{filepath.Join(dir, "cq-myapp-web.container")}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	sys := newMockSystemdClient()
	sys.stopErr["cq-myapp-web.service"] = errors.New("stop failed")
	o := newTestOrchestrator("myapp", dir, state, sys)

	err := o.Stop(nil, false)
	if err == nil || !strings.Contains(err.Error(), "stop failed") {
		t.Errorf("expected stop error propagated, got %v", err)
	}
}

func TestStop_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.Stop(nil, false)
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Restart
// ---------------------------------------------------------------------------

func TestRestart_CallsRestartUnitForEachContainer(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "cq-myapp-web.container"),
	}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, state, sys)

	if err := o.Restart(nil, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sys.restarted) != 1 || sys.restarted[0] != "cq-myapp-web.service" {
		t.Errorf("expected web restarted, got %v", sys.restarted)
	}
}

func TestRestart_PropagatesRestartError(t *testing.T) {
	dir := t.TempDir()
	files := []string{filepath.Join(dir, "cq-myapp-web.container")}
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, files),
	})
	sys := newMockSystemdClient()
	sys.restartErr["cq-myapp-web.service"] = errors.New("restart failed")
	o := newTestOrchestrator("myapp", dir, state, sys)

	err := o.Restart(nil, false)
	if err == nil || !strings.Contains(err.Error(), "restart failed") {
		t.Errorf("expected restart error propagated, got %v", err)
	}
}

func TestRestart_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.Restart(nil, false)
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}
