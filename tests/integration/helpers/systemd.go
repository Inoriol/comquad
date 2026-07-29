//go:build integration

package helpers

import (
    "context"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

// systemdTargetDir returns the systemd quadlet target directory for the current user.
// Root: /etc/containers/systemd  Non-root: ~/.config/containers/systemd
func systemdTargetDir(t *testing.T) string {
    t.Helper()
    if os.Getuid() == 0 {
        return "/etc/containers/systemd"
    }
    home, err := os.UserHomeDir()
    if err != nil {
        t.Fatalf("could not determine home dir: %v", err)
    }
    return filepath.Join(home, ".config", "containers", "systemd")
}

// SystemdAvailable returns true if the systemd D-Bus socket is accessible
// and systemctl can communicate with the running systemd instance (system or
// user, depending on whether we're running as root).
func SystemdAvailable(t *testing.T) bool {
    t.Helper()
    // Check if the D-Bus socket exists first. "systemctl is-system-running"
    // returns "offline" even when systemd is not running at all, so we
    // need a positive signal that systemd is actually reachable.
    if os.Getuid() == 0 {
        if _, err := os.Stat("/run/dbus/system_bus_socket"); err != nil {
            return false
        }
    } else {
        rdir := os.Getenv("XDG_RUNTIME_DIR")
        if rdir == "" {
            return false
        }
        if _, err := os.Stat(rdir + "/bus"); err != nil {
            return false
        }
    }

    args := []string{"systemctl"}
    if os.Getuid() != 0 {
        args = append(args, "--user")
    }
    args = append(args, "is-system-running")
    // is-system-running exits 0 only for "running", non-zero for "degraded"
    // and all other states. We check the output string, not the exit code.
    // Use a timeout: systemctl can hang if the socket exists but systemd
    // is not actually responding to D-Bus requests.
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    out, _ := exec.CommandContext(ctx, args[0], args[1:]...).Output()
    status := strings.TrimSpace(string(out))
    return status == "running" || status == "degraded" || status == "starting" || status == "initializing"
}

// SkipIfSystemdUnavailable skips the test when systemd is not reachable.
func SkipIfSystemdUnavailable(t *testing.T) {
    t.Helper()
    if !SystemdAvailable(t) {
        t.Skip("systemd is not available — skipping test that requires systemd")
    }
}

// QuadletFileExists returns true if a quadlet file with the given basename exists
// in the systemd target directory (e.g. "cq-myproject-web.container").
func QuadletFileExists(t *testing.T, basename string) bool {
    t.Helper()
    path := filepath.Join(systemdTargetDir(t), basename)
    info, err := os.Stat(path)
    return err == nil && info.Mode().IsRegular()
}

// WaitForUnit polls systemctl until the unit reaches the expected ActiveState
// or the timeout is exceeded. Uses --user when rootless is true.
func WaitForUnit(t *testing.T, unit, wantState string, rootless bool, timeout time.Duration) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        state := UnitActiveState(t, unit, rootless)
        if state == wantState {
            return
        }
        time.Sleep(500 * time.Millisecond)
    }
    t.Fatalf("unit %s did not reach state %q within %s (last state: %s)",
        unit, wantState, timeout, UnitActiveState(t, unit, rootless))
}

// UnitActiveState returns the ActiveState of a systemd unit.
func UnitActiveState(t *testing.T, unit string, rootless bool) string {
    t.Helper()
    args := []string{"systemctl"}
    if rootless {
        args = append(args, "--user")
    }
    args = append(args, "show", "-p", "ActiveState", "--value", unit)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    out, err := exec.CommandContext(ctx, args[0], args[1:]...).Output()
    if err != nil {
        return "unknown"
    }
    return strings.TrimSpace(string(out))
}

// AssertUnitActive fails the test if the unit is not active.
func AssertUnitActive(t *testing.T, unit string, rootless bool) {
    t.Helper()
    WaitForUnit(t, unit, "active", rootless, 30*time.Second)
}

// AssertUnitInactive fails the test if the unit is still active after timeout.
func AssertUnitInactive(t *testing.T, unit string, rootless bool) {
    t.Helper()
    WaitForUnit(t, unit, "inactive", rootless, 30*time.Second)
}

// UnitExists returns true if systemd knows about the unit at all.
func UnitExists(t *testing.T, unit string, rootless bool) bool {
    t.Helper()
    args := []string{"systemctl"}
    if rootless {
        args = append(args, "--user")
    }
    args = append(args, "cat", unit)
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    err := exec.CommandContext(ctx, args[0], args[1:]...).Run()
    return err == nil
}
