package orchestrator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"comquad/internal/deploy"
)

func TestDown_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.Down(false, false)
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

func TestDown_StateError(t *testing.T) {
	o := newTestOrchestratorWithStateErr("myapp", t.TempDir(), errors.New("state unavailable"))
	err := o.Down(false, false)
	if err == nil || !strings.Contains(err.Error(), "state unavailable") {
		t.Errorf("expected state error, got %v", err)
	}
}

func TestDown_SystemdConnectionError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	o := newTestOrchestratorWithSystemdErr("myapp", dir, state, errors.New("dbus gone"))

	err := o.Down(false, false)
	if err == nil || !strings.Contains(err.Error(), "dbus gone") {
		t.Errorf("expected dbus error, got %v", err)
	}
}

func TestDown_StopsUnitsBeforeRemovingFiles(t *testing.T) {
	dir := t.TempDir()
	containerPath := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerPath, "[Container]\nImage=nginx\n")

	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{containerPath}),
	})
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, state, sys)

	if err := o.Down(false, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sys.stoppedUnits) != 1 || sys.stoppedUnits[0] != "cq-myapp-web.service" {
		t.Errorf("expected web unit to be stopped, got %v", sys.stoppedUnits)
	}
}

func TestDown_RemovesQuadletFiles(t *testing.T) {
	dir := t.TempDir()
	containerPath := filepath.Join(dir, "cq-myapp-web.container")
	networkPath := filepath.Join(dir, "cq-myapp-default.network")
	writeFile(t, containerPath, "[Container]\nImage=nginx\n")
	writeFile(t, networkPath, "[Network]\n")

	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{containerPath, networkPath}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	if err := o.Down(false, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, path := range []string{containerPath, networkPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected file %q to be removed", path)
		}
	}
}

func TestDown_ReloadsDaemonAfterFileRemoval(t *testing.T) {
	dir := t.TempDir()
	containerPath := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerPath, "[Container]\nImage=nginx\n")

	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{containerPath}),
	})
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, state, sys)

	if err := o.Down(false, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sys.reloadCalls) == 0 {
		t.Error("expected ReloadDaemon to be called")
	}
}

func TestDown_UnregistersProjectFromState(t *testing.T) {
	dir := t.TempDir()
	containerPath := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerPath, "[Container]\nImage=nginx\n")

	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{containerPath}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	if err := o.Down(false, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := state.projects["myapp"]; exists {
		t.Error("expected project to be unregistered from state")
	}
}

func TestDown_ReloadDaemonError_PropagatesError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	sys := newMockSystemdClient()
	sys.reloadErr = errors.New("reload failed")
	o := newTestOrchestrator("myapp", dir, state, sys)

	err := o.Down(false, false)
	if err == nil || !strings.Contains(err.Error(), "reload failed") {
		t.Errorf("expected reload error, got %v", err)
	}
}

func TestDown_AlreadyRemovedFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	// File doesn't exist on disk — Down should not error on missing files.
	missingPath := filepath.Join(dir, "cq-myapp-web.container")

	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{missingPath}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	if err := o.Down(false, false); err != nil {
		t.Errorf("Down should tolerate already-missing quadlet files, got: %v", err)
	}
}

func TestDown_UnregisterError_PropagatesError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	state.saveErr = errors.New("disk full")
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.Down(false, false)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Errorf("expected unregister error, got %v", err)
	}
}

func TestDown_StopsNetworkAndVolumeUnits(t *testing.T) {
	dir := t.TempDir()
	containerPath := filepath.Join(dir, "cq-myapp-web.container")
	networkPath := filepath.Join(dir, "cq-myapp-default.network")
	volumePath := filepath.Join(dir, "cq-myapp-data.volume")
	writeFile(t, containerPath, "[Container]\nImage=nginx\n")
	writeFile(t, networkPath, "[Network]\n")
	writeFile(t, volumePath, "[Volume]\n")

	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{containerPath, networkPath, volumePath}),
	})
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, state, sys)

	if err := o.Down(false, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedUnits := []string{"cq-myapp-web.service", "cq-myapp-default-network.service", "cq-myapp-data-volume.service"}
	for _, expected := range expectedUnits {
		found := false
		for _, stopped := range sys.stoppedUnits {
			if stopped == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected unit %q to be stopped, got %v", expected, sys.stoppedUnits)
		}
	}
}

func TestStopUnits_StopsOnlyContainers(t *testing.T) {
	dir := t.TempDir()
	containerFile := writeContainerFile(t, dir, "cq-myapp-web.container")
	networkFile := filepath.Join(dir, "cq-myapp-default.network")
	volumeFile := filepath.Join(dir, "cq-myapp-data.volume")
	writeFile(t, networkFile, "[Network]\n")
	writeFile(t, volumeFile, "[Volume]\n")

	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, newMockStateStore(nil), sys)

	err := o.stopUnits(sys, []string{containerFile, networkFile, volumeFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sys.stoppedUnits) != 1 {
		t.Fatalf("expected 1 stopped unit, got %d: %v", len(sys.stoppedUnits), sys.stoppedUnits)
	}
	if sys.stoppedUnits[0] != "cq-myapp-web.service" {
		t.Errorf("expected 'cq-myapp-web.service', got %q", sys.stoppedUnits[0])
	}
}

func TestStopUnits_NoContainerFiles(t *testing.T) {
	dir := t.TempDir()
	networkFile := filepath.Join(dir, "cq-myapp-default.network")
	writeFile(t, networkFile, "[Network]\n")

	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, newMockStateStore(nil), sys)

	err := o.stopUnits(sys, []string{networkFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sys.stoppedUnits) != 0 {
		t.Errorf("expected 0 stopped units, got %d", len(sys.stoppedUnits))
	}
}

func TestStopUnits_MultipleContainers(t *testing.T) {
	dir := t.TempDir()
	webFile := writeContainerFile(t, dir, "cq-myapp-web.container")
	dbFile := writeContainerFile(t, dir, "cq-myapp-db.container")

	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, newMockStateStore(nil), sys)

	err := o.stopUnits(sys, []string{webFile, dbFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sys.stoppedUnits) != 2 {
		t.Fatalf("expected 2 stopped units, got %d", len(sys.stoppedUnits))
	}
}

func TestStopUnits_PropagatesError(t *testing.T) {
	dir := t.TempDir()
	containerFile := writeContainerFile(t, dir, "cq-myapp-web.container")

	sys := newMockSystemdClient()
	sys.stopErr = map[string]error{
		"cq-myapp-web.service": fmt.Errorf("unit does not exist"),
	}
	o := newTestOrchestrator("myapp", dir, newMockStateStore(nil), sys)

	err := o.stopUnits(sys, []string{containerFile})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to stop unit") {
		t.Errorf("expected error to contain 'failed to stop unit', got %q", err.Error())
	}
}

func TestStopUnits_EmptyProjectFiles(t *testing.T) {
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	err := o.stopUnits(sys, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
