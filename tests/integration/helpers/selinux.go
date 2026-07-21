//go:build integration

package helpers

import (
 "os"
 "strings"
 "testing"
)

// SELinuxEnforcing returns true if SELinux is currently in enforcing mode.
// Mirrors the exact detection logic comquad uses internally
// (reading /sys/fs/selinux/enforce) so tests reflect real behavior.
func SELinuxEnforcing(t *testing.T) bool {
 t.Helper()
 data, err := os.ReadFile("/sys/fs/selinux/enforce")
 if err != nil {
  // File absent means SELinux is not present on this kernel/container
  return false
 }
 return strings.TrimSpace(string(data)) == "1"
}

// SELinuxPresent returns true if the SELinux filesystem is mounted at all,
// regardless of enforcement mode. Useful to distinguish "disabled" from
// "permissive" vs "enforcing".
func SELinuxPresent(t *testing.T) bool {
 t.Helper()
 _, err := os.Stat("/sys/fs/selinux")
 return err == nil
}

// SkipIfSELinuxAbsent skips the test if SELinux is not present on the system.
func SkipIfSELinuxAbsent(t *testing.T) {
	t.Helper()
	if !SELinuxPresent(t) {
		t.Skip("SELinux not present on this system — skipping SELinux test")
	}
}

// SkipIfSELinuxNotEnabled skips the test if SELinux is not enabled (enforcing or
// permissive). This matches the exact detection logic comquad uses for SELinux
// label injection — presence alone is not enough; the enforcement file must
// contain "0" or "1".
func SkipIfSELinuxNotEnabled(t *testing.T) {
	t.Helper()
	SkipIfSELinuxAbsent(t)
	if !SELinuxEnabled(t) {
		t.Skip("SELinux present but not enabled — skipping SELinux test")
	}
}

// SELinuxEnabled returns true if SELinux is enabled (enforcing or permissive).
// Mirrors the exact detection logic comquad uses internally.
func SELinuxEnabled(t *testing.T) bool {
	t.Helper()
	data, err := os.ReadFile("/sys/fs/selinux/enforce")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "0" || strings.TrimSpace(string(data)) == "1"
}

// SkipIfSELinuxNotEnforcing skips the test if SELinux is present but not
// in enforcing mode. Some label injection behaviors only matter when
// enforcing, but the `,z` injection in comquad triggers on presence alone.
func SkipIfSELinuxNotEnforcing(t *testing.T) {
 t.Helper()
 SkipIfSELinuxAbsent(t)
 if !SELinuxEnforcing(t) {
  t.Skip("SELinux present but not enforcing — skipping enforcement test")
 }
}
