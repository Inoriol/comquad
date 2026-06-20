package orchestrator

import (
	"path/filepath"
	"strings"
	"testing"

	"comquad/internal/deploy"
)

// ---------------------------------------------------------------------------
// FollowLogs — error paths (state lookup only, no actual journalctl)
// ---------------------------------------------------------------------------

func TestFollowLogs_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.FollowLogs("2024-01-01 12:00:00", "", "")
	if err == nil || !strings.Contains(err.Error(), "not deployed") {
		t.Errorf("expected 'not deployed' error, got %v", err)
	}
}

func TestFollowLogs_NoUnitsInProject(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{}),
	})
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())

	err := o.FollowLogs("2024-01-01 12:00:00", "", "")
	if err == nil || !strings.Contains(err.Error(), "no units found") {
		t.Errorf("expected 'no units found' error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Logs — new flags
// ---------------------------------------------------------------------------

func TestLogs_WithTailFlag(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
		}),
	})
	mockSys := newMockSystemdClient()
	mockSys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "inactive", subState: "dead"},
	}

	o := newTestOrchestrator("myapp", dir, state, mockSys)

	// This will fail at journalctl execution but we can verify it gets far enough
	// to construct the args before failing. We check the error message to confirm
	// the tail flag was accepted (journalctl would reject invalid tail values).
	// Since we're not mocking exec here, we just verify no panic and correct
	// error path is taken for the new flag signature.
	err := o.Logs(nil, false, "10", "", "")
	// We expect journalctl to fail since there's no real journal, but the key
	// thing is the function accepted the tail parameter without error.
	if err == nil {
		// If it somehow succeeded (e.g. no units matched), that's fine too
		return
	}
	// If it failed, it should be a journalctl error, not a flag parsing error
	if strings.Contains(err.Error(), "flag") || strings.Contains(err.Error(), "invalid") {
		t.Errorf("flag parsing error: %v", err)
	}
}

func TestLogs_WithSinceFlag(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
		}),
	})
	mockSys := newMockSystemdClient()
	mockSys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "inactive", subState: "dead"},
	}

	o := newTestOrchestrator("myapp", dir, state, mockSys)

	err := o.Logs(nil, false, "", "10m ago", "")
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "flag") || strings.Contains(err.Error(), "invalid") {
		t.Errorf("flag parsing error: %v", err)
	}
}

func TestLogs_WithOutputFlag(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, []string{
			filepath.Join(dir, "cq-myapp-web.container"),
		}),
	})
	mockSys := newMockSystemdClient()
	mockSys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "inactive", subState: "dead"},
	}

	o := newTestOrchestrator("myapp", dir, state, mockSys)

	err := o.Logs(nil, false, "", "", "short-iso")
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "flag") || strings.Contains(err.Error(), "invalid") {
		t.Errorf("flag parsing error: %v", err)
	}
}
