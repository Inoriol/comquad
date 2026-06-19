package transpile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeFakePodlet writes a shell script to dir that acts as a fake podlet binary
// and returns its path. The script behaviour is controlled by the body argument,
// which must be valid bash inserted after the shebang line.
func makeFakePodlet(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "podlet")
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake podlet: %v", err)
	}
	return path
}

// withPATH prepends dirs to the front of PATH for the duration of the test,
// keeping the existing PATH entries so system utilities (cat, touch, etc.)
// remain available inside fake scripts. For the "missing podlet" test an
// empty-only path is set explicitly.
func withPATH(t *testing.T, dirs ...string) {
	t.Helper()
	old := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", old) })
	os.Setenv("PATH", strings.Join(append(dirs, old), ":"))
}

// withEmptyPATH replaces PATH with a single empty temp dir, preventing any
// real binary from being found.
func withEmptyPATH(t *testing.T) {
	t.Helper()
	old := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", old) })
	os.Setenv("PATH", t.TempDir())
}

// -----------------------------------------------------------------------------
// NewPodletRunner
// -----------------------------------------------------------------------------

func TestNewPodletRunner_ErrorWhenPodletMissing(t *testing.T) {
	// Replace PATH entirely with an empty dir so podlet cannot be found.
	withEmptyPATH(t)

	_, err := NewPodletRunner(t.TempDir())
	if err == nil {
		t.Fatal("expected error when podlet is not in PATH, got nil")
	}
	if !strings.Contains(err.Error(), "podlet not found in PATH") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewPodletRunner_SuccessWhenPodletPresent(t *testing.T) {
	binDir := t.TempDir()
	makeFakePodlet(t, binDir, "exit 0")
	withPATH(t, binDir)

	runner, err := NewPodletRunner(t.TempDir())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
	if runner.PodletPath == "" {
		t.Error("expected PodletPath to be set")
	}
	if !filepath.IsAbs(runner.PodletPath) {
		t.Errorf("expected PodletPath to be absolute, got %q", runner.PodletPath)
	}
}

func TestNewPodletRunner_StoresTempDir(t *testing.T) {
	binDir := t.TempDir()
	makeFakePodlet(t, binDir, "exit 0")
	withPATH(t, binDir)

	tmpDir := t.TempDir()
	runner, err := NewPodletRunner(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.TempDir != tmpDir {
		t.Errorf("expected TempDir %q, got %q", tmpDir, runner.TempDir)
	}
}

// -----------------------------------------------------------------------------
// Transpile — happy path
// -----------------------------------------------------------------------------

func TestTranspile_PassesOutputDirFlag(t *testing.T) {
	binDir := t.TempDir()
	outDir := t.TempDir()

	// Script echoes its arguments to a file so we can inspect them.
	argFile := filepath.Join(outDir, "args.txt")
	makeFakePodlet(t, binDir, `echo "$@" > `+argFile)
	withPATH(t, binDir)

	runner, err := NewPodletRunner(outDir)
	if err != nil {
		t.Fatalf("NewPodletRunner: %v", err)
	}

	if err := runner.Transpile([]byte("services: {}")); err != nil {
		t.Fatalf("Transpile failed: %v", err)
	}

	argsRaw, err := os.ReadFile(argFile)
	if err != nil {
		t.Fatalf("could not read args file: %v", err)
	}
	args := strings.TrimSpace(string(argsRaw))

	if !strings.Contains(args, "-f") {
		t.Errorf("expected -f flag to be passed, got args: %q", args)
	}
	if !strings.Contains(args, outDir) {
		t.Errorf("expected output dir %q in args, got: %q", outDir, args)
	}
	if !strings.Contains(args, "compose") {
		t.Errorf("expected 'compose' subcommand in args, got: %q", args)
	}
}

func TestTranspile_ReceivesYAMLOnStdin(t *testing.T) {
	binDir := t.TempDir()
	outDir := t.TempDir()

	// Script writes its stdin to a file so we can inspect what was piped.
	stdinFile := filepath.Join(outDir, "stdin.txt")
	makeFakePodlet(t, binDir, `cat > `+stdinFile)
	withPATH(t, binDir)

	runner, err := NewPodletRunner(outDir)
	if err != nil {
		t.Fatalf("NewPodletRunner: %v", err)
	}

	input := []byte("services:\n  web:\n    image: nginx\n")
	if err := runner.Transpile(input); err != nil {
		t.Fatalf("Transpile failed: %v", err)
	}

	got, err := os.ReadFile(stdinFile)
	if err != nil {
		t.Fatalf("could not read stdin file: %v", err)
	}
	if string(got) != string(input) {
		t.Errorf("stdin mismatch:\n  want: %q\n  got:  %q", input, got)
	}
}

func TestTranspile_WritesOutputFiles(t *testing.T) {
	binDir := t.TempDir()
	outDir := t.TempDir()

	// Script creates two quadlet files in the output directory, simulating podlet.
	makeFakePodlet(t, binDir,
		`touch "$2/web.container" "$2/default.network"`,
	)
	withPATH(t, binDir)

	runner, err := NewPodletRunner(outDir)
	if err != nil {
		t.Fatalf("NewPodletRunner: %v", err)
	}

	if err := runner.Transpile([]byte("services: {}")); err != nil {
		t.Fatalf("Transpile failed: %v", err)
	}

	for _, name := range []string{"web.container", "default.network"} {
		path := filepath.Join(outDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected output file %q to exist", name)
		}
	}
}

// -----------------------------------------------------------------------------
// Transpile — error paths
// -----------------------------------------------------------------------------

func TestTranspile_ReturnsErrorOnNonZeroExit(t *testing.T) {
	binDir := t.TempDir()
	outDir := t.TempDir()

	makeFakePodlet(t, binDir, `echo "invalid compose file" >&2; exit 1`)
	withPATH(t, binDir)

	runner, err := NewPodletRunner(outDir)
	if err != nil {
		t.Fatalf("NewPodletRunner: %v", err)
	}

	err = runner.Transpile([]byte("not: valid: compose"))
	if err == nil {
		t.Fatal("expected error on non-zero exit, got nil")
	}
	if !strings.Contains(err.Error(), "podlet execution failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestTranspile_IncludesStderrInError(t *testing.T) {
	binDir := t.TempDir()
	outDir := t.TempDir()

	makeFakePodlet(t, binDir, `echo "detailed error message" >&2; exit 2`)
	withPATH(t, binDir)

	runner, err := NewPodletRunner(outDir)
	if err != nil {
		t.Fatalf("NewPodletRunner: %v", err)
	}

	err = runner.Transpile([]byte("bad input"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "detailed error message") {
		t.Errorf("expected stderr in error message, got: %v", err)
	}
}

func TestTranspile_ReturnsErrorOnBadBinary(t *testing.T) {
	// Construct a runner directly with a non-existent binary path,
	// bypassing NewPodletRunner so we can test cmd.Start() failure.
	runner := &PodletRunner{
		PodletPath: "/nonexistent/path/to/podlet",
		TempDir:    t.TempDir(),
	}

	err := runner.Transpile([]byte("services: {}"))
	if err == nil {
		t.Fatal("expected error for non-existent binary, got nil")
	}
	if !strings.Contains(err.Error(), "failed to start podlet") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestTranspile_EmptyInput(t *testing.T) {
	binDir := t.TempDir()
	outDir := t.TempDir()

	// Script exits successfully regardless of input.
	makeFakePodlet(t, binDir, "exit 0")
	withPATH(t, binDir)

	runner, err := NewPodletRunner(outDir)
	if err != nil {
		t.Fatalf("NewPodletRunner: %v", err)
	}

	// Empty input should not cause a panic or a pipe error — just run and exit.
	if err := runner.Transpile([]byte{}); err != nil {
		t.Errorf("unexpected error for empty input: %v", err)
	}
}
