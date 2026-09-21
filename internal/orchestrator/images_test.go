package orchestrator

import (
	"strings"
	"testing"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
	"github.com/Inoriol/comquad/internal/reconcile"
)

func TestSetPolicyOnImageUnits_SetsPolicyOnAllImageUnits(t *testing.T) {
	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitImage,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionImage, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx"}},
				}},
			},
		},
		{
			Type: c2q.UnitImage,
			Name: "cq-myapp-db",
			Sections: []c2q.Section{
				{Name: c2q.SectionImage, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/postgres"}},
				}},
			},
		},
	}

	setPolicyOnImageUnits(units, PullAlways)

	for _, unit := range units {
		if unit.Type != c2q.UnitImage {
			continue
		}
		policy := getDirective(unit, c2q.SectionImage, "Policy")
		if policy != "always" {
			t.Errorf("expected Policy=always on %s, got %q", unit.Name, policy)
		}
	}
}

func TestSetPolicyOnImageUnits_OverridesExistingPolicy(t *testing.T) {
	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitImage,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionImage, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx"}},
					{Key: "Policy", Values: []string{"missing"}},
				}},
			},
		},
	}

	setPolicyOnImageUnits(units, PullNever)

	policy := getDirective(units[0], c2q.SectionImage, "Policy")
	if policy != "never" {
		t.Errorf("expected Policy=never, got %q", policy)
	}
}

func TestSetPolicyOnImageUnits_SkipsNonImageUnits(t *testing.T) {
	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"nginx"}},
				}},
			},
		},
	}

	setPolicyOnImageUnits(units, PullAlways)

	policy := getDirective(units[0], c2q.SectionContainer, "Policy")
	if policy != "" {
		t.Errorf("expected no Policy on container unit, got %q", policy)
	}
}

func TestHandleImages_StartsImageUnits(t *testing.T) {
	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-web-image.service", activeState: "inactive", subState: "dead"},
	}

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	projectFiles := []string{"/fake/cq-myapp-web.image"}
	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitImage,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionImage, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx"}},
				}},
			},
		},
	}

	err := o.handleImages(projectFiles, units, "missing")
	if err != nil {
		t.Fatalf("handleImages failed: %v", err)
	}

	if len(sys.startedUnits) != 1 {
		t.Errorf("expected 1 started unit, got %d", len(sys.startedUnits))
	}
	if sys.startedUnits[0] != "cq-myapp-web-image.service" {
		t.Errorf("expected cq-myapp-web-image.service to be started, got %s", sys.startedUnits[0])
	}
}

func TestHandleImages_SkipsBuildUnits(t *testing.T) {
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	projectFiles := []string{"/fake/cq-myapp-web.image"}
	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitImage,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionImage, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"myapp-web:latest"}},
				}},
			},
		},
		{
			Type: c2q.UnitBuild,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionBuild, Directives: []c2q.Directive{
					{Key: "ImageTag", Values: []string{"myapp-web:latest"}},
				}},
			},
		},
	}

	err := o.handleImages(projectFiles, units, "always")
	if err != nil {
		t.Fatalf("handleImages failed: %v", err)
	}

	if len(sys.startedUnits) != 0 {
		t.Errorf("expected 0 started units (build should be skipped), got %d", len(sys.startedUnits))
	}
}

func TestHandleImages_PullAlwaysStopsAndStarts(t *testing.T) {
	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-web-image.service", activeState: "active", subState: "exited"},
	}

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	projectFiles := []string{"/fake/cq-myapp-web.image"}
	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitImage,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionImage, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx"}},
				}},
			},
		},
	}

	err := o.handleImages(projectFiles, units, "always")
	if err != nil {
		t.Fatalf("handleImages failed: %v", err)
	}

	if len(sys.stoppedUnits) != 1 {
		t.Errorf("expected 1 stopped unit, got %d", len(sys.stoppedUnits))
	}
	if len(sys.startedUnits) != 1 {
		t.Errorf("expected 1 started unit, got %d", len(sys.startedUnits))
	}
}

func TestHandleImages_PullMissingSkipsAlreadyPulled(t *testing.T) {
	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-web-image.service", activeState: "inactive", subState: "exited"},
	}

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	projectFiles := []string{"/fake/cq-myapp-web.image"}
	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitImage,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionImage, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx"}},
				}},
			},
		},
	}

	err := o.handleImages(projectFiles, units, "missing")
	if err != nil {
		t.Fatalf("handleImages failed: %v", err)
	}

	if len(sys.startedUnits) != 0 {
		t.Errorf("expected 0 started units (already pulled), got %d: %v", len(sys.startedUnits), sys.startedUnits)
	}
}

func TestHandleBuilds_StartsBuildUnits(t *testing.T) {
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	projectFiles := []string{"/fake/cq-myapp-web.build"}

	err := o.handleBuilds(projectFiles)
	if err != nil {
		t.Fatalf("handleBuilds failed: %v", err)
	}

	if len(sys.stoppedUnits) != 1 {
		t.Errorf("expected 1 stopped unit, got %d", len(sys.stoppedUnits))
	}
	if len(sys.startedUnits) != 1 {
		t.Errorf("expected 1 started unit, got %d", len(sys.startedUnits))
	}
	if sys.startedUnits[0] != "cq-myapp-web-build.service" {
		t.Errorf("expected cq-myapp-web-build.service to be started, got %s", sys.startedUnits[0])
	}
}

func TestHandleBuilds_IgnoresNonBuildFiles(t *testing.T) {
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	projectFiles := []string{"/fake/cq-myapp-web.container", "/fake/cq-myapp-web.image"}

	err := o.handleBuilds(projectFiles)
	if err != nil {
		t.Fatalf("handleBuilds failed: %v", err)
	}

	if len(sys.startedUnits) != 0 {
		t.Errorf("expected 0 started units, got %d", len(sys.startedUnits))
	}
}

func TestPrepareUnits_StopsRemovedAndReloads(t *testing.T) {
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	res := reconcileResultWithRemoved([]string{"/fake/cq-myapp-old.container"})

	err := o.prepareUnits(nil, res)
	if err != nil {
		t.Fatalf("prepareUnits failed: %v", err)
	}

	if len(sys.stoppedUnits) != 1 {
		t.Errorf("expected 1 stopped unit, got %d", len(sys.stoppedUnits))
	}
	if sys.stoppedUnits[0] != "cq-myapp-old.service" {
		t.Errorf("expected cq-myapp-old.service to be stopped, got %s", sys.stoppedUnits[0])
	}
	if len(sys.reloadCalls) != 1 {
		t.Errorf("expected 1 reload call, got %d", len(sys.reloadCalls))
	}
}

func TestStartContainers_StartsContainerUnits(t *testing.T) {
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	projectFiles := []string{"/fake/cq-myapp-web.container"}
	res := reconcileResultWithCreated([]string{"/fake/cq-myapp-web.container"})

	err := o.startContainers(projectFiles, res)
	if err != nil {
		t.Fatalf("startContainers failed: %v", err)
	}

	if len(sys.startedUnits) != 1 {
		t.Errorf("expected 1 started unit, got %d", len(sys.startedUnits))
	}
	if sys.startedUnits[0] != "cq-myapp-web.service" {
		t.Errorf("expected cq-myapp-web.service to be started, got %s", sys.startedUnits[0])
	}
}

func TestStartContainers_RestartsChangedUnits(t *testing.T) {
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), sys)

	projectFiles := []string{"/fake/cq-myapp-web.container"}
	res := reconcileResultWithChanged([]string{"/fake/cq-myapp-web.container"})

	err := o.startContainers(projectFiles, res)
	if err != nil {
		t.Fatalf("startContainers failed: %v", err)
	}

	if len(sys.restarted) != 1 {
		t.Errorf("expected 1 restarted unit, got %d", len(sys.restarted))
	}
	if sys.restarted[0] != "cq-myapp-web.service" {
		t.Errorf("expected cq-myapp-web.service to be restarted, got %s", sys.restarted[0])
	}
}

func TestHasBuildUnitForName(t *testing.T) {
	units := []c2q.QuadletUnit{
		{Type: c2q.UnitBuild, Name: "cq-myapp-web"},
		{Type: c2q.UnitImage, Name: "cq-myapp-db"},
	}

	if !hasBuildUnitForName(units, "cq-myapp-web") {
		t.Error("expected hasBuildUnitForName to return true for cq-myapp-web")
	}
	if hasBuildUnitForName(units, "cq-myapp-db") {
		t.Error("expected hasBuildUnitForName to return false for cq-myapp-db")
	}
	if hasBuildUnitForName(units, "cq-myapp-nonexistent") {
		t.Error("expected hasBuildUnitForName to return false for nonexistent unit")
	}
}

func TestHasImageUnitForName(t *testing.T) {
	units := []c2q.QuadletUnit{
		{Type: c2q.UnitImage, Name: "cq-myapp-web"},
		{Type: c2q.UnitBuild, Name: "cq-myapp-db"},
	}

	if !hasImageUnitForName(units, "cq-myapp-web") {
		t.Error("expected hasImageUnitForName to return true for cq-myapp-web")
	}
	if hasImageUnitForName(units, "cq-myapp-db") {
		t.Error("expected hasImageUnitForName to return false for cq-myapp-db")
	}
}

func reconcileResultWithRemoved(removed []string) reconcile.Result {
	return reconcile.Result{
		Removed: removed,
	}
}

func reconcileResultWithCreated(created []string) reconcile.Result {
	return reconcile.Result{
		Created: created,
	}
}

func reconcileResultWithChanged(changed []string) reconcile.Result {
	return reconcile.Result{
		Changed: changed,
	}
}

func TestParsePullStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected PullStrategy
		hasError bool
	}{
		{"always", PullAlways, false},
		{"ALWAYS", PullAlways, false},
		{"missing", PullMissing, false},
		{"MISSING", PullMissing, false},
		{"never", PullNever, false},
		{"NEVER", PullNever, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParsePullStrategy(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestPrintDryRun_ShowsBuildUnits(t *testing.T) {
	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitBuild,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionBuild, Directives: []c2q.Directive{
					{Key: "ImageTag", Values: []string{"myapp-web:latest"}},
				}},
			},
		},
	}
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "always", dryRunPlan(t, t.TempDir(), units))
	})

	if !strings.Contains(out, "[build]") {
		t.Errorf("expected [build] label in output, got:\n%s", out)
	}
	if !strings.Contains(out, "myapp-web:latest") {
		t.Errorf("expected image tag in output, got:\n%s", out)
	}
}
