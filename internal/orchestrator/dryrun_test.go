package orchestrator

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
	"github.com/Inoriol/comquad/internal/deploy"
)

var captureStdoutMu sync.Mutex

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	captureStdoutMu.Lock()
	defer captureStdoutMu.Unlock()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("captureStdout: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()
	return buf.String()
}

func makeTestUnits() []c2q.QuadletUnit {
	return []c2q.QuadletUnit{
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx"}},
					{Key: "Label", Values: []string{"com.comquad.project=myapp"}},
				}},
				{Name: c2q.SectionInstall, Directives: []c2q.Directive{
					{Key: "WantedBy", Values: []string{"default.target"}},
				}},
			},
		},
	}
}

func makeBuildTestUnits() []c2q.QuadletUnit {
	return []c2q.QuadletUnit{
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"myapp-web:latest"}},
				}},
				{Name: c2q.SectionInstall, Directives: []c2q.Directive{
					{Key: "WantedBy", Values: []string{"default.target"}},
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
}

func makeMixedTestUnits() []c2q.QuadletUnit {
	return []c2q.QuadletUnit{
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"myapp-web:latest"}},
				}},
				{Name: c2q.SectionInstall, Directives: []c2q.Directive{
					{Key: "WantedBy", Values: []string{"default.target"}},
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
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-db",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/postgres:15"}},
				}},
				{Name: c2q.SectionInstall, Directives: []c2q.Directive{
					{Key: "WantedBy", Values: []string{"default.target"}},
				}},
			},
		},
	}
}

func makeMultiTestUnits() []c2q.QuadletUnit {
	return []c2q.QuadletUnit{
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx"}},
				}},
				{Name: c2q.SectionInstall, Directives: []c2q.Directive{
					{Key: "WantedBy", Values: []string{"default.target"}},
				}},
			},
		},
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-db",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/postgres:15"}},
				}},
				{Name: c2q.SectionInstall, Directives: []c2q.Directive{
					{Key: "WantedBy", Values: []string{"default.target"}},
				}},
			},
		},
		{
			Type: c2q.UnitNetwork,
			Name: "cq-myapp-default",
			Sections: []c2q.Section{
				{Name: c2q.SectionNetwork, Directives: []c2q.Directive{}},
			},
		},
	}
}

func makeImageRefTestUnits() []c2q.QuadletUnit {
	return []c2q.QuadletUnit{
		{
			Type: c2q.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionContainer, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"cq-myapp-web.image"}},
				}},
				{Name: c2q.SectionInstall, Directives: []c2q.Directive{
					{Key: "WantedBy", Values: []string{"default.target"}},
				}},
			},
		},
		{
			Type: c2q.UnitImage,
			Name: "cq-myapp-web",
			Sections: []c2q.Section{
				{Name: c2q.SectionImage, Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx:latest"}},
				}},
			},
		},
	}
}

func TestPrintDryRun_PrintsProjectAndTargetDir(t *testing.T) {
	units := makeTestUnits()
	targetDir := t.TempDir()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		if err := o.printDryRun(units, targetDir, "missing"); err != nil {
			t.Errorf("printDryRun error: %v", err)
		}
	})

	if !strings.Contains(out, "myapp") {
		t.Errorf("expected project name in output, got:\n%s", out)
	}
	if !strings.Contains(out, targetDir) {
		t.Errorf("expected target dir in output, got:\n%s", out)
	}
}

func TestPrintDryRun_ShowsTargetPathForEachFile(t *testing.T) {
	units := makeTestUnits()
	targetDir := "/fake/systemd/target"
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, targetDir, "missing")
	})

	expectedTarget := filepath.Join(targetDir, "cq-myapp-web.container")
	if !strings.Contains(out, expectedTarget) {
		t.Errorf("expected target path %q in output, got:\n%s", expectedTarget, out)
	}
}

func TestPrintDryRun_ShowsFileContent(t *testing.T) {
	units := makeTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "missing")
	})

	if !strings.Contains(out, "[Container]") {
		t.Errorf("expected file content [Container] in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Image=docker.io/library/nginx") {
		t.Errorf("expected Image= line in output, got:\n%s", out)
	}
}

func TestPrintDryRun_PrintsFileCount(t *testing.T) {
	units := makeTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "missing")
	})

	if !strings.Contains(out, "1 quadlet file(s)") {
		t.Errorf("expected '1 quadlet file(s)' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_PrintsDryRunCompleteSummary(t *testing.T) {
	units := makeTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "missing")
	})

	if !strings.Contains(out, "Dry run complete") {
		t.Errorf("expected 'Dry run complete' summary, got:\n%s", out)
	}
	if !strings.Contains(out, "nothing was written") {
		t.Errorf("expected 'nothing was written' in summary, got:\n%s", out)
	}
}

func TestPrintDryRun_ImagePullNeverReported(t *testing.T) {
	units := makeTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "never")
	})

	if !strings.Contains(out, "pull skipped: never") {
		t.Errorf("expected 'pull skipped: never' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_ImagePullAlwaysReported(t *testing.T) {
	units := makeTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "always")
	})

	if !strings.Contains(out, "would pull: always") {
		t.Errorf("expected 'would pull: always' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_MultipleFilesAllShown(t *testing.T) {
	units := makeMultiTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, "/fake/target", "missing")
	})

	for _, name := range []string{"cq-myapp-web.container", "cq-myapp-db.container", "cq-myapp-default.network"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected %q in dry-run output, got:\n%s", name, out)
		}
	}
}

func TestPrintDryRun_InvalidPullStrategy(t *testing.T) {
	units := makeTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	err := o.printDryRun(units, t.TempDir(), "badstrategy")
	if err == nil {
		t.Error("expected error for invalid pull strategy, got nil")
	}
}

func TestUp_DryRun_DoesNotWriteToTargetDir(t *testing.T) {
	dir := t.TempDir()
	targetDir := t.TempDir()
	writeFile(t, filepath.Join(dir, "compose.yaml"), "services:\n  web:\n    image: nginx\n")

	state := newMockStateStore(nil)
	o := &Orchestrator{
		projectName: "myapp",
		cwd:         dir,
		newState: func() (deploy.StateStore, error) { return state, nil },
		newSystemd: func() (deploy.SystemdClient, error) {
			return newMockSystemdClient(), nil
		},
	}

	captureStdout(t, func() {
		o.Up("missing", false, true)
	})

	if len(state.projects) != 0 {
		t.Errorf("dry-run must not register state, got %v", state.projects)
	}

	entries, _ := os.ReadDir(targetDir)
	if len(entries) != 0 {
		t.Errorf("dry-run must not write to target dir, found %d files", len(entries))
	}
}

func TestUp_DryRun_NoComposeFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())
	o.cwd = dir

	err := o.Up("missing", false, true)
	if err == nil || !strings.Contains(err.Error(), "no compose file found") {
		t.Errorf("expected 'no compose file found', got %v", err)
	}
}

func TestPrintDryRun_BuildContainerShowsBuildLabel(t *testing.T) {
	units := makeBuildTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "always")
	})

	if !strings.Contains(out, "[build]") {
		t.Errorf("expected [build] label in output for built container, got:\n%s", out)
	}
	if !strings.Contains(out, "would be built locally") {
		t.Errorf("expected 'would be built locally' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_BuildContainerSkipsPullLabels(t *testing.T) {
	units := makeBuildTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "always")
	})

	if strings.Contains(out, "[image]") {
		t.Errorf("expected no [image] pull label for built container, got:\n%s", out)
	}
}

func TestPrintDryRun_MixedBuildAndImageContainers(t *testing.T) {
	units := makeMixedTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "always")
	})

	if !strings.Contains(out, "[build]") {
		t.Errorf("expected [build] label for built web container, got:\n%s", out)
	}
	if !strings.Contains(out, "[image]") {
		t.Errorf("expected [image] label for db container, got:\n%s", out)
	}
}

func TestPrintDryRun_ResolvesImageRef(t *testing.T) {
	units := makeImageRefTestUnits()
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(units, t.TempDir(), "missing")
	})

	if !strings.Contains(out, "docker.io/library/nginx:latest") {
		t.Errorf("expected resolved image name in output, got:\n%s", out)
	}
}

func TestHasBuildFile_Match(t *testing.T) {
	files := []string{
		"/path/to/cq-myapp-web.container",
		"/path/to/cq-myapp-web.build",
		"/path/to/cq-myapp-db.container",
	}

	if !hasBuildFile(files, "/path/to/cq-myapp-web.container") {
		t.Error("expected true for container with matching .build file")
	}
}

func TestHasBuildFile_NoMatch(t *testing.T) {
	files := []string{
		"/path/to/cq-myapp-web.container",
		"/path/to/cq-myapp-db.container",
	}

	if hasBuildFile(files, "/path/to/cq-myapp-web.container") {
		t.Error("expected false for container without .build file")
	}
}

func TestHasBuildFile_EmptyList(t *testing.T) {
	var files []string
	if hasBuildFile(files, "/path/to/cq-myapp-web.container") {
		t.Error("expected false for empty file list")
	}
}
