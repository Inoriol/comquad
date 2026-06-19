package orchestrator

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"comquad/internal/deploy"
	"comquad/internal/preprocess"
)

// captureStdout redirects os.Stdout for the duration of fn and returns the
// captured output. The original stdout is always restored.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
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

// makePreviewDir creates a temp directory with a minimal cooked quadlet file
// and returns the dir path and the file's absolute path.
func makePreviewDir(t *testing.T) (previewDir string, containerFile string) {
	t.Helper()
	dir := t.TempDir()
	content := "[Container]\nImage=docker.io/library/nginx\nLabel=com.comquad.project=myapp\n\n[Install]\nWantedBy=default.target\n"
	path := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, path, content)
	return dir, path
}

// ---------------------------------------------------------------------------
// printDryRun — output structure
// ---------------------------------------------------------------------------

func TestPrintDryRun_PrintsProjectAndTargetDir(t *testing.T) {
	previewDir, containerFile := makePreviewDir(t)
	targetDir := t.TempDir()

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		if err := o.printDryRun([]string{containerFile}, previewDir, targetDir, nil, "missing"); err != nil {
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
	previewDir, containerFile := makePreviewDir(t)
	targetDir := "/fake/systemd/target"

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, previewDir, targetDir, nil, "missing")
	})

	expectedTarget := filepath.Join(targetDir, "cq-myapp-web.container")
	if !strings.Contains(out, expectedTarget) {
		t.Errorf("expected target path %q in output, got:\n%s", expectedTarget, out)
	}
}

func TestPrintDryRun_ShowsFileContent(t *testing.T) {
	previewDir, containerFile := makePreviewDir(t)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, previewDir, t.TempDir(), nil, "missing")
	})

	if !strings.Contains(out, "[Container]") {
		t.Errorf("expected file content [Container] in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Image=docker.io/library/nginx") {
		t.Errorf("expected Image= line in output, got:\n%s", out)
	}
}

func TestPrintDryRun_PrintsFileCount(t *testing.T) {
	previewDir, containerFile := makePreviewDir(t)

	// Add a second file
	networkFile := filepath.Join(previewDir, "cq-myapp-default.network")
	writeFile(t, networkFile, "[Network]\n")

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile, networkFile}, previewDir, t.TempDir(), nil, "missing")
	})

	if !strings.Contains(out, "2 quadlet file(s)") {
		t.Errorf("expected '2 quadlet file(s)' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_PrintsDryRunCompleteSummary(t *testing.T) {
	previewDir, containerFile := makePreviewDir(t)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, previewDir, t.TempDir(), nil, "missing")
	})

	if !strings.Contains(out, "Dry run complete") {
		t.Errorf("expected 'Dry run complete' summary, got:\n%s", out)
	}
	if !strings.Contains(out, "nothing was written") {
		t.Errorf("expected 'nothing was written' in summary, got:\n%s", out)
	}
}

func TestPrintDryRun_ImagePullNeverReported(t *testing.T) {
	previewDir, containerFile := makePreviewDir(t)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, previewDir, t.TempDir(), nil, "never")
	})

	if !strings.Contains(out, "pull skipped: never") {
		t.Errorf("expected 'pull skipped: never' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_ImagePullAlwaysReported(t *testing.T) {
	previewDir, containerFile := makePreviewDir(t)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, previewDir, t.TempDir(), nil, "always")
	})

	if !strings.Contains(out, "would pull: always") {
		t.Errorf("expected 'would pull: always' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_BuildServiceReported(t *testing.T) {
	previewDir := t.TempDir()
	// Container file with a build-generated image tag (contains ":")
	content := "[Container]\nImage=myapp-web:latest\n\n[Install]\nWantedBy=default.target\n"
	containerFile := filepath.Join(previewDir, "cq-myapp-web.container")
	writeFile(t, containerFile, content)

	buildInfo := map[string]*preprocess.BuildInfo{
		"web": {
			Context:    "/some/context",
			Dockerfile: "Dockerfile",
			Service:    "web",
		},
	}

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, previewDir, t.TempDir(), buildInfo, "missing")
	})

	// Should mention the service and the build context
	if !strings.Contains(out, "web") {
		t.Errorf("expected service name 'web' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "would build") {
		// Image may already exist locally; either "would build" or "already exists" must appear
		if !strings.Contains(out, "already exists") {
			t.Errorf("expected build info in output, got:\n%s", out)
		}
	}
}

func TestPrintDryRun_MultipleFilesAllShown(t *testing.T) {
	previewDir := t.TempDir()

	files := []string{}
	for _, name := range []string{
		"cq-myapp-web.container",
		"cq-myapp-db.container",
		"cq-myapp-default.network",
	} {
		path := filepath.Join(previewDir, name)
		writeFile(t, path, fmt.Sprintf("[Container]\nLabel=name=%s\n", name))
		files = append(files, path)
	}

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(files, previewDir, "/fake/target", nil, "missing")
	})

	for _, name := range []string{"cq-myapp-web.container", "cq-myapp-db.container", "cq-myapp-default.network"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected %q in dry-run output, got:\n%s", name, out)
		}
	}
}

func TestPrintDryRun_InvalidPullStrategy(t *testing.T) {
	previewDir, containerFile := makePreviewDir(t)
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	err := o.printDryRun([]string{containerFile}, previewDir, t.TempDir(), nil, "badstrategy")
	if err == nil {
		t.Error("expected error for invalid pull strategy, got nil")
	}
}

// ---------------------------------------------------------------------------
// Up with dryRun=true — integration guard
// ---------------------------------------------------------------------------

func TestUp_DryRun_DoesNotWriteToTargetDir(t *testing.T) {
	// This test requires podlet in PATH. Skip gracefully if not present.
	if _, err := os.LookupEnv("COMQUAD_TEST_WITH_PODLET"); err {
		t.Skip("set COMQUAD_TEST_WITH_PODLET=1 to run tests requiring podlet")
	}

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
	// Override target dir resolution by pointing at our temp dir
	_ = targetDir

	captureStdout(t, func() {
		o.Up(false, "missing", false, true)
	})

	// State must NOT have been registered
	if len(state.projects) != 0 {
		t.Errorf("dry-run must not register state, got %v", state.projects)
	}

	// Target dir must remain empty (nothing copied there)
	entries, _ := os.ReadDir(targetDir)
	if len(entries) != 0 {
		t.Errorf("dry-run must not write to target dir, found %d files", len(entries))
	}
}

func TestUp_DryRun_NoComposeFileReturnsError(t *testing.T) {
	dir := t.TempDir() // empty dir — no compose file
	state := newMockStateStore(nil)
	o := newTestOrchestrator("myapp", dir, state, newMockSystemdClient())
	o.cwd = dir

	err := o.Up(false, "missing", false, true)
	if err == nil || !strings.Contains(err.Error(), "no compose file found") {
		t.Errorf("expected 'no compose file found', got %v", err)
	}
}
