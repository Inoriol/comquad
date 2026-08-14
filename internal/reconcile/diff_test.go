package reconcile

import (
	"strings"
	"testing"
)

func TestUnifiedDiff_ModifyLine(t *testing.T) {
	old := "[Container]\nImage=web.image\n"
	new := "[Container]\nImage=web-v2.image\n"
	got := unifiedDiff("cq-app-web.container", old, new)
	if !strings.Contains(got, "--- a/cq-app-web.container") {
		t.Errorf("missing old header: %q", got)
	}
	if !strings.Contains(got, "+++ b/cq-app-web.container") {
		t.Errorf("missing new header: %q", got)
	}
	if !strings.Contains(got, "-Image=web.image") || !strings.Contains(got, "+Image=web-v2.image") {
		t.Errorf("missing changed lines: %q", got)
	}
}

func TestUnifiedDiff_Created(t *testing.T) {
	got := unifiedDiff("cq-app-web.container", "", "[Container]\nImage=web.image\n")
	if !strings.Contains(got, "--- /dev/null") {
		t.Errorf("expected /dev/null old header: %q", got)
	}
	if !strings.Contains(got, "+Image=web.image") {
		t.Errorf("expected added line: %q", got)
	}
	if strings.Contains(got, "-") && !strings.Contains(got, "+++") {
		t.Errorf("unexpected removal in created diff: %q", got)
	}
}

func TestUnifiedDiff_Removed(t *testing.T) {
	got := unifiedDiff("cq-app-db.container", "[Container]\nImage=db.image\n", "")
	if !strings.Contains(got, "+++ /dev/null") {
		t.Errorf("expected /dev/null new header: %q", got)
	}
	if !strings.Contains(got, "-Image=db.image") {
		t.Errorf("expected removed line: %q", got)
	}
}

func TestUnifiedDiff_NoChange(t *testing.T) {
	got := unifiedDiff("cq-app-web.container", "[Container]\nImage=web.image\n", "[Container]\nImage=web.image\n")
	if got != "" {
		t.Errorf("expected empty diff, got %q", got)
	}
}

func TestUnifiedDiff_TrailingNewlineInsensitive(t *testing.T) {
	got := unifiedDiff("x", "a\nb\n", "a\nb")
	if got != "" {
		t.Errorf("trailing newline should not produce diff, got %q", got)
	}
}

func TestPlan_HasChanges(t *testing.T) {
	empty := Plan{}
	if empty.HasChanges() {
		t.Error("empty plan should have no changes")
	}
	changed := Plan{Files: []FilePlan{{Status: StatusChanged}}}
	if !changed.HasChanges() {
		t.Error("changed plan should report changes")
	}
	removed := Plan{Files: []FilePlan{{Status: StatusRemoved}}}
	if !removed.HasChanges() {
		t.Error("removed plan should report changes")
	}
	unchanged := Plan{Files: []FilePlan{{Status: StatusUnchanged}}}
	if unchanged.HasChanges() {
		t.Error("unchanged plan should report no changes")
	}
}
