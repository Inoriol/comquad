//go:build integration

package helpers

import (
	"os/exec"
	"strings"
	"testing"
)

// ExecInContainer runs a command inside a running container and returns stdout.
func ExecInContainer(t *testing.T, container, command string) string {
	t.Helper()
	ctx := exec.Command("podman", "exec", container, "sh", "-c", command)
	out, err := ctx.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("podman exec %q in %q failed (exit %d): %s",
				command, container, exitErr.ExitCode(), string(exitErr.Stderr))
		}
		t.Fatalf("podman exec %q in %q failed: %v", command, container, err)
	}
	return strings.TrimSpace(string(out))
}

// AssertSecretContent reads a secret file inside a container and checks its value.
func AssertSecretContent(t *testing.T, container, secretPath, expected string) {
	t.Helper()
	actual := ExecInContainer(t, container, "cat "+secretPath)
	if actual != expected {
		t.Fatalf("secret at %s in container %s: expected %q, got %q",
			secretPath, container, expected, actual)
	}
}

// AssertSecretExists checks that a secret file exists inside a container.
func AssertSecretExists(t *testing.T, container, secretPath string) {
	t.Helper()
	ExecInContainer(t, container, "test -f "+secretPath)
}

// AssertSecretServiceActive checks that a secret systemd service is active (exited successfully).
func AssertSecretServiceActive(t *testing.T, project, secretName string, rootless bool) {
	t.Helper()
	unitName := "cq-" + project + "-" + secretName + "-secret.service"
	AssertUnitActive(t, unitName, rootless)
}

// AssertSecretServiceGone checks that a secret systemd service no longer exists.
func AssertSecretServiceGone(t *testing.T, project, secretName string, rootless bool) {
	t.Helper()
	unitName := "cq-" + project + "-" + secretName + "-secret.service"
	if UnitExists(t, unitName, rootless) {
		t.Fatalf("expected secret service %s to be removed but it still exists", unitName)
	}
}

// AssertSecretQuadletFileExists checks that a .cq-secret quadlet file exists.
func AssertSecretQuadletFileExists(t *testing.T, project, secretName string) {
	t.Helper()
	basename := "cq-" + project + "-" + secretName + ".cq-secret"
	if !QuadletFileExists(t, basename) {
		t.Fatalf("expected quadlet file %s to exist", basename)
	}
}

// AssertSecretQuadletFileGone checks that a .cq-secret quadlet file no longer exists.
func AssertSecretQuadletFileGone(t *testing.T, project, secretName string) {
	t.Helper()
	basename := "cq-" + project + "-" + secretName + ".cq-secret"
	if QuadletFileExists(t, basename) {
		t.Fatalf("expected quadlet file %s to be removed but it still exists", basename)
	}
}
