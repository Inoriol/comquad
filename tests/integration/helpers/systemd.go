//go:build integration

package helpers

import (
    "os/exec"
    "strings"
    "testing"
    "time"
)

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

    out, err := exec.Command(args[0], args[1:]...).Output()
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
    err := exec.Command(args[0], args[1:]...).Run()
    return err == nil
}
