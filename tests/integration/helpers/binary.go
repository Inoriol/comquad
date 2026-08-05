//go:build integration

package helpers

import (
    "bytes"
    "fmt"
    "os"
    "os/exec"
    "strings"
    "testing"
    "time"
)

// CQResult holds the result of a comquad invocation.
type CQResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
}

// Comquad invokes the comquad binary with the given arguments.
// It resolves the binary from the CQ_BINARY env var, falling back to
// "comquad" on PATH. The working directory is set to workDir when non-empty.
func Comquad(t *testing.T, workDir string, args ...string) CQResult {
    t.Helper()

    bin := os.Getenv("CQ_BINARY")
    if bin == "" {
        bin = "comquad"
    }

    cmd := exec.Command(bin, args...)
    if workDir != "" {
        cmd.Dir = workDir
    }

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()

    result := CQResult{
        Stdout:   strings.TrimSpace(stdout.String()),
        Stderr:   strings.TrimSpace(stderr.String()),
        ExitCode: 0,
    }

    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            result.ExitCode = exitErr.ExitCode()
        }
    }

    t.Logf("[comquad %s] exit=%d\nstdout: %s\nstderr: %s",
        strings.Join(args, " "), result.ExitCode, result.Stdout, result.Stderr)

    return result
}

// MustSucceed calls Comquad and fails the test immediately if exit code != 0.
func MustSucceed(t *testing.T, workDir string, args ...string) CQResult {
    t.Helper()
    result := Comquad(t, workDir, args...)
    if result.ExitCode != 0 {
        t.Fatalf("comquad %s failed (exit %d):\nstdout: %s\nstderr: %s",
            strings.Join(args, " "), result.ExitCode, result.Stdout, result.Stderr)
    }
    return result
}

// MustFail calls Comquad and fails the test if exit code == 0.
func MustFail(t *testing.T, workDir string, args ...string) CQResult {
    t.Helper()
    result := Comquad(t, workDir, args...)
    if result.ExitCode == 0 {
        t.Fatalf("expected comquad %s to fail but it succeeded:\nstdout: %s",
            strings.Join(args, " "), result.Stdout)
    }
    return result
}

// ProjectName generates a unique project name for a test to avoid
// collisions between parallel runs.
func ProjectName(t *testing.T) string {
	t.Helper()
	// sanitize test name: lowercase, replace slashes and spaces with dashes
	name := strings.ToLower(t.Name())
	name = strings.NewReplacer("/", "-", " ", "-", "_", "-").Replace(name)
	// keep it short — systemd unit names have limits
	if len(name) > 40 {
		name = name[:40]
	}
	return fmt.Sprintf("cqt-%s", name)
}

// WaitForLogs polls comquad logs until output is produced or the timeout is exceeded.
func WaitForLogs(t *testing.T, dir string, project string, args ...string) CQResult {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	logArgs := append([]string{"logs", "--name", project}, args...)
	for time.Now().Before(deadline) {
		result := Comquad(t, dir, logArgs...)
		if result.ExitCode == 0 && (result.Stdout != "" || result.Stderr != "") {
			return result
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("logs for project %s did not produce output within 30s", project)
	return CQResult{}
}
