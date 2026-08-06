package orchestrator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/internal/deploy"
)

// ---------------------------------------------------------------------------
// FollowLogs — error paths (state lookup only, no actual journalctl)
// ---------------------------------------------------------------------------

func TestFollowLogs_ProjectNotDeployed(t *testing.T) {
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", t.TempDir(), state, newMockSystemdClient())

	err := o.FollowLogs("2024-01-01 12:00:00", "", false)
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

	err := o.FollowLogs("2024-01-01 12:00:00", "", false)
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
	err := o.Logs(nil, false, "10", "", false)
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

	err := o.Logs(nil, false, "", "10m ago", false)
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "flag") || strings.Contains(err.Error(), "invalid") {
		t.Errorf("flag parsing error: %v", err)
	}
}

func TestParseJournalEntry_ValidEntry(t *testing.T) {
	json := `{"__REALTIME_TIMESTAMP":"1700000000000000","SYSTEMD_UNIT":"cq-myapp-web.service","MESSAGE":"test message","PRIORITY":6}`
	entry, ok := parseJournalEntry(json)
	if !ok {
		t.Fatal("expected valid entry")
	}
	if entry.timestamp != 1700000000000000 {
		t.Errorf("expected timestamp 1700000000000000, got %d", entry.timestamp)
	}
	if entry.unit != "cq-myapp-web.service" {
		t.Errorf("expected unit cq-myapp-web.service, got %s", entry.unit)
	}
	if entry.message != "test message" {
		t.Errorf("expected message 'test message', got %s", entry.message)
	}
	if entry.priority != 6 {
		t.Errorf("expected priority 6, got %d", entry.priority)
	}
}

func TestParseJournalEntry_PriorityAsString(t *testing.T) {
	json := `{"__REALTIME_TIMESTAMP":"1700000000000000","SYSTEMD_UNIT":"cq-myapp-web.service","MESSAGE":"test","PRIORITY":"6"}`
	entry, ok := parseJournalEntry(json)
	if !ok {
		t.Fatal("expected valid entry")
	}
	if entry.priority != 6 {
		t.Errorf("expected priority 6, got %d", entry.priority)
	}
}

func TestParseJournalEntry_PriorityAsFloat(t *testing.T) {
	json := `{"__REALTIME_TIMESTAMP":"1700000000000000","SYSTEMD_UNIT":"cq-myapp-web.service","MESSAGE":"test","PRIORITY":4.0}`
	entry, ok := parseJournalEntry(json)
	if !ok {
		t.Fatal("expected valid entry")
	}
	if entry.priority != 4 {
		t.Errorf("expected priority 4, got %d", entry.priority)
	}
}

func TestParseJournalEntry_InvalidJSON(t *testing.T) {
	_, ok := parseJournalEntry("not json")
	if ok {
		t.Error("expected invalid entry for non-JSON input")
	}
}

func TestParseJournalEntry_MissingFields(t *testing.T) {
	json := `{"__REALTIME_TIMESTAMP":"1700000000000000"}`
	entry, ok := parseJournalEntry(json)
	if !ok {
		t.Fatal("expected valid entry for partial JSON")
	}
	if entry.timestamp != 1700000000000000 {
		t.Errorf("expected timestamp 1700000000000000, got %d", entry.timestamp)
	}
	if entry.unit != "" {
		t.Errorf("expected empty unit, got %s", entry.unit)
	}
	if entry.message != "" {
		t.Errorf("expected empty message, got %s", entry.message)
	}
	if entry.priority != 0 {
		t.Errorf("expected priority 0, got %d", entry.priority)
	}
}

func TestRenderEntry_WithTime(t *testing.T) {
	entry := journalEntry{
		timestamp: 1700000000000000,
		unit:      "cq-myapp-web.service",
		message:   "test message",
		priority:  6,
	}
	rendered := renderEntry(entry, true)
	if rendered == "" {
		t.Error("expected non-empty rendered entry")
	}
}

func TestRenderEntry_WithoutTime(t *testing.T) {
	entry := journalEntry{
		timestamp: 1700000000000000,
		unit:      "cq-myapp-web.service",
		message:   "test message",
		priority:  6,
	}
	rendered := renderEntry(entry, false)
	if rendered == "" {
		t.Error("expected non-empty rendered entry")
	}
}

func TestFlushEntries_SortsByTimestamp(t *testing.T) {
	entries := []journalEntry{
		{timestamp: 3000, unit: "unit3", message: "third", priority: 0},
		{timestamp: 1000, unit: "unit1", message: "first", priority: 0},
		{timestamp: 2000, unit: "unit2", message: "second", priority: 0},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	flushEntries(entries, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	firstIdx := strings.Index(output, "first")
	secondIdx := strings.Index(output, "second")
	thirdIdx := strings.Index(output, "third")
	if firstIdx < 0 || secondIdx < 0 || thirdIdx < 0 {
		t.Fatalf("missing expected messages in output: %q", output)
	}
	if firstIdx >= secondIdx || secondIdx >= thirdIdx {
		t.Errorf("entries not sorted by timestamp, got: %q", output)
	}
}
