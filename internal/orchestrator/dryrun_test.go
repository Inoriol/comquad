package orchestrator

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

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

// makePreviewDir creates a temp directory with a minimal cooked quadlet file
// and returns the dir path, the file's absolute path, and its content for the fileContents map.
func makePreviewDir(t *testing.T) (previewDir string, containerFile string, fileContent string) {
	t.Helper()
	dir := t.TempDir()
	content := "[Container]\nImage=docker.io/library/nginx\nLabel=com.comquad.project=myapp\n\n[Install]\nWantedBy=default.target\n"
	path := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, path, content)
	return dir, path, content
}

func makeFileContentMap(paths ...string) map[string]string {
	m := make(map[string]string)
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		m[p] = string(data)
	}
	return m
}

// ---------------------------------------------------------------------------
// printDryRun — output structure
// ---------------------------------------------------------------------------

func TestPrintDryRun_PrintsProjectAndTargetDir(t *testing.T) {
		previewDir, containerFile, _ := makePreviewDir(t)
	targetDir := t.TempDir()

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		if err := o.printDryRun([]string{containerFile}, makeFileContentMap(containerFile), previewDir, targetDir, "missing"); err != nil {
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
		previewDir, containerFile, _ := makePreviewDir(t)
	targetDir := "/fake/systemd/target"

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, makeFileContentMap(containerFile), previewDir, targetDir, "missing")
	})

	expectedTarget := filepath.Join(targetDir, "cq-myapp-web.container")
	if !strings.Contains(out, expectedTarget) {
		t.Errorf("expected target path %q in output, got:\n%s", expectedTarget, out)
	}
}

func TestPrintDryRun_ShowsFileContent(t *testing.T) {
		previewDir, containerFile, _ := makePreviewDir(t)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, makeFileContentMap(containerFile), previewDir, t.TempDir(), "missing")
	})

	if !strings.Contains(out, "[Container]") {
		t.Errorf("expected file content [Container] in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Image=docker.io/library/nginx") {
		t.Errorf("expected Image= line in output, got:\n%s", out)
	}
}

func TestPrintDryRun_PrintsFileCount(t *testing.T) {
		previewDir, containerFile, _ := makePreviewDir(t)

	// Add a second file
	networkFile := filepath.Join(previewDir, "cq-myapp-default.network")
	writeFile(t, networkFile, "[Network]\n")

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile, networkFile}, makeFileContentMap(containerFile, networkFile), previewDir, t.TempDir(), "missing")
	})

	if !strings.Contains(out, "2 quadlet file(s)") {
		t.Errorf("expected '2 quadlet file(s)' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_PrintsDryRunCompleteSummary(t *testing.T) {
		previewDir, containerFile, _ := makePreviewDir(t)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, makeFileContentMap(containerFile), previewDir, t.TempDir(), "missing")
	})

	if !strings.Contains(out, "Dry run complete") {
		t.Errorf("expected 'Dry run complete' summary, got:\n%s", out)
	}
	if !strings.Contains(out, "nothing was written") {
		t.Errorf("expected 'nothing was written' in summary, got:\n%s", out)
	}
}

func TestPrintDryRun_ImagePullNeverReported(t *testing.T) {
		previewDir, containerFile, _ := makePreviewDir(t)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, makeFileContentMap(containerFile), previewDir, t.TempDir(), "never")
	})

	if !strings.Contains(out, "pull skipped: never") {
		t.Errorf("expected 'pull skipped: never' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_ImagePullAlwaysReported(t *testing.T) {
		previewDir, containerFile, _ := makePreviewDir(t)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun([]string{containerFile}, makeFileContentMap(containerFile), previewDir, t.TempDir(), "always")
	})

	if !strings.Contains(out, "would pull: always") {
		t.Errorf("expected 'would pull: always' in output, got:\n%s", out)
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
		o.printDryRun(files, makeFileContentMap(files...), previewDir, "/fake/target", "missing")
	})

	for _, name := range []string{"cq-myapp-web.container", "cq-myapp-db.container", "cq-myapp-default.network"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected %q in dry-run output, got:\n%s", name, out)
		}
	}
}

func TestPrintDryRun_InvalidPullStrategy(t *testing.T) {
		previewDir, containerFile, _ := makePreviewDir(t)
	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	err := o.printDryRun([]string{containerFile}, makeFileContentMap(containerFile), previewDir, t.TempDir(), "badstrategy")
	if err == nil {
		t.Error("expected error for invalid pull strategy, got nil")
	}
}

// ---------------------------------------------------------------------------
// Up with dryRun=true — integration guard
// ---------------------------------------------------------------------------

func TestUp_DryRun_DoesNotWriteToTargetDir(t *testing.T) {
	if _, err := exec.LookPath("podlet"); err != nil {
		t.Skip("podlet not available")
	}

	dir := t.TempDir()
	targetDir := t.TempDir()
	writeFile(t, filepath.Join(dir, "compose.yaml"), "services:\n  web:\n    image: nginx\n")

	state := newMockStateStore(nil)
	// Use manual Orchestrator construction so we can override newState/newSystemd
	// while still hitting the real resolveTargetDir/transpile/cook paths.
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

	err := o.Up("missing", false, true)
	if err == nil || !strings.Contains(err.Error(), "no compose file found") {
		t.Errorf("expected 'no compose file found', got %v", err)
	}
}

func TestPrintDryRun_BuildContainerShowsBuildLabel(t *testing.T) {
	previewDir := t.TempDir()

	containerContent := "[Container]\nImage=myapp-web:latest\n\n[Install]\nWantedBy=default.target\n"
	containerFile := filepath.Join(previewDir, "cq-myapp-web.container")
	writeFile(t, containerFile, containerContent)

	buildContent := "[Build]\nImageTag=myapp-web:latest\nFile=" + filepath.Join(previewDir, "Dockerfile") + "\n"
	buildFile := filepath.Join(previewDir, "cq-myapp-web.build")
	writeFile(t, buildFile, buildContent)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(
			[]string{containerFile, buildFile},
			makeFileContentMap(containerFile, buildFile),
			previewDir,
			t.TempDir(),
			"always",
		)
	})

	if !strings.Contains(out, "[build]") {
		t.Errorf("expected [build] label in output for built container, got:\n%s", out)
	}
	if !strings.Contains(out, "would be built locally") {
		t.Errorf("expected 'would be built locally' in output, got:\n%s", out)
	}
}

func TestPrintDryRun_BuildContainerSkipsPullLabels(t *testing.T) {
	previewDir := t.TempDir()

	containerContent := "[Container]\nImage=myapp-web:latest\n\n[Install]\nWantedBy=default.target\n"
	containerFile := filepath.Join(previewDir, "cq-myapp-web.container")
	writeFile(t, containerFile, containerContent)

	buildFile := filepath.Join(previewDir, "cq-myapp-web.build")
	writeFile(t, buildFile, "[Build]\nImageTag=myapp-web:latest\n")

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(
			[]string{containerFile, buildFile},
			makeFileContentMap(containerFile, buildFile),
			previewDir,
			t.TempDir(),
			"always",
		)
	})

	if strings.Contains(out, "[image]") {
		t.Errorf("expected no [image] pull label for built container, got:\n%s", out)
	}
}

func TestPrintDryRun_MixedBuildAndImageContainers(t *testing.T) {
	previewDir := t.TempDir()

	webContent := "[Container]\nImage=myapp-web:latest\n\n[Install]\nWantedBy=default.target\n"
	webFile := filepath.Join(previewDir, "cq-myapp-web.container")
	writeFile(t, webFile, webContent)

	webBuild := filepath.Join(previewDir, "cq-myapp-web.build")
	writeFile(t, webBuild, "[Build]\nImageTag=myapp-web:latest\n")

	dbContent := "[Container]\nImage=docker.io/library/postgres:15\n\n[Install]\nWantedBy=default.target\n"
	dbFile := filepath.Join(previewDir, "cq-myapp-db.container")
	writeFile(t, dbFile, dbContent)

	o := newTestOrchestrator("myapp", t.TempDir(), newMockStateStore(nil), newMockSystemdClient())

	out := captureStdout(t, func() {
		o.printDryRun(
			[]string{webFile, webBuild, dbFile},
			makeFileContentMap(webFile, webBuild, dbFile),
			previewDir,
			t.TempDir(),
			"always",
		)
	})

	if !strings.Contains(out, "[build]") {
		t.Errorf("expected [build] label for built web container, got:\n%s", out)
	}
	if !strings.Contains(out, "[image]") {
		t.Errorf("expected [image] label for db container, got:\n%s", out)
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
