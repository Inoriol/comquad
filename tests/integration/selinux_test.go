//go:build integration

package integration

import (
 "fmt"
 "strings"
 "testing"

 "comquad/tests/integration/helpers"
)

// composeWithBindMount returns a compose file with a bind-mounted host path,
// an explicit :ro bind mount, an explicit :rw bind mount, and a named volume —
// covering all four injection cases described in the architecture.
func composeWithBindMount(project, hostPath string) string {
 return `name: ` + project + `
services:
  app:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    volumes:
      - ` + hostPath + `:/data
      - ` + hostPath + `:/data-ro:ro
      - ` + hostPath + `:/data-rw:rw
      - appdata:/named
volumes:
  appdata:
`
}

// composeWithZAlreadyPresent returns a compose where :z is already set,
// verifying comquad does not double-inject.
func composeWithZAlreadyPresent(project, hostPath string) string {
 return `name: ` + project + `
services:
  app:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    volumes:
      - ` + hostPath + `:/data:z
      - ` + hostPath + `:/data-Z:Z
`
}

// --- Quadlet file content tests ---
// These verify the Cook stage injects ,z into the quadlet file itself,
// which happens when SELinux is enabled (enforcing or permissive).

func TestSELinux_QuadletFile_ZInjected_WhenSELinuxPresent(t *testing.T) {
	helpers.SkipIfSELinuxNotEnabled(t)

 project := helpers.ProjectName(t)
 hostPath := t.TempDir()
 dir, _ := helpers.WriteCompose(t, composeWithBindMount(project, hostPath))

 // Use dry-run so we can inspect the quadlet file content without deploying
 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // All three bind mount variants must have ,z appended
 for _, expected := range []string{
  "/data:z",
  "/data-ro:ro,z",
  "/data-rw:rw,z",
 } {
  if !strings.Contains(result.Stdout, expected) {
   t.Fatalf("SELinux: expected %q in quadlet output, got:\n%s", expected, result.Stdout)
  }
 }
}

func TestSELinux_QuadletFile_ZNotInjected_WhenSELinuxAbsent(t *testing.T) {
	if helpers.SELinuxEnabled(t) {
		t.Skip("SELinux is enabled on this system — skipping absence test")
	}

 project := helpers.ProjectName(t)
 hostPath := t.TempDir()
 dir, _ := helpers.WriteCompose(t, composeWithBindMount(project, hostPath))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // Without SELinux, no ,z should be injected
 for _, unexpected := range []string{":z", ":ro,z", ":rw,z"} {
  if strings.Contains(result.Stdout, unexpected) {
   t.Fatalf("no SELinux: unexpected %q found in quadlet output:\n%s",
    unexpected, result.Stdout)
  }
 }
}

func TestSELinux_QuadletFile_ZIdempotent(t *testing.T) {
 helpers.SkipIfSELinuxAbsent(t)

 project := helpers.ProjectName(t)
 hostPath := t.TempDir()
 dir, _ := helpers.WriteCompose(t, composeWithZAlreadyPresent(project, hostPath))

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // :z already present — must not become :z,z or :Z,z
 for _, bad := range []string{":z,z", ":Z,z", ":z,Z"} {
  if strings.Contains(result.Stdout, bad) {
   t.Fatalf("SELinux z-injection is not idempotent, found %q in output:\n%s",
    bad, result.Stdout)
  }
 }

 // Original labels must still be present exactly once
 for _, expected := range []string{"/data:z", "/data-Z:Z"} {
  count := strings.Count(result.Stdout, expected)
  if count != 1 {
   t.Fatalf("expected exactly one occurrence of %q, found %d:\n%s",
    expected, count, result.Stdout)
  }
 }
}

func TestSELinux_QuadletFile_NamedVolume_NoZInjection(t *testing.T) {
	helpers.SkipIfSELinuxNotEnabled(t)

 project := helpers.ProjectName(t)
 hostPath := t.TempDir()
 dir, _ := helpers.WriteCompose(t, composeWithBindMount(project, hostPath))

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // Named volumes (appdata:/named) must NOT get :z — only bind mounts do
 // The named volume line in the quadlet will reference the volume unit,
 // not a host path, so it should never contain :z
 lines := strings.Split(result.Stdout, "\n")
 for _, line := range lines {
  if strings.Contains(line, "appdata") && strings.Contains(line, ":z") {
   t.Fatalf("named volume line must not have :z injected: %q", line)
  }
 }
}

// --- Runtime mount option tests (SELinux enforcing required) ---
// These verify the actual running container has the correct mount labels
// applied by the kernel, not just that the quadlet file was written correctly.

func TestSELinux_Runtime_MountHasZLabel(t *testing.T) {
 helpers.SkipIfSELinuxNotEnforcing(t)

 project := helpers.ProjectName(t)
 hostPath := t.TempDir()
 dir, _ := helpers.WriteCompose(t, composeWithBindMount(project, hostPath))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 containerName := fmt.Sprintf("%s-app", project)
 unitName := fmt.Sprintf("cq-%s-app.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // The plain bind mount (/data) must have the z relabeling option at runtime
 helpers.AssertMountHasOption(t, containerName, "/data", "z")
}

func TestSELinux_Runtime_ReadOnlyMountHasZLabel(t *testing.T) {
 helpers.SkipIfSELinuxNotEnforcing(t)

 project := helpers.ProjectName(t)
 hostPath := t.TempDir()
 dir, _ := helpers.WriteCompose(t, composeWithBindMount(project, hostPath))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 containerName := fmt.Sprintf("%s-app", project)
 unitName := fmt.Sprintf("cq-%s-app.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // :ro,z — both ro and z must be present
 helpers.AssertMountHasOption(t, containerName, "/data-ro", "ro")
 helpers.AssertMountHasOption(t, containerName, "/data-ro", "z")
}

func TestSELinux_Runtime_NoZLabel_WhenSELinuxAbsent(t *testing.T) {
 if helpers.SELinuxPresent(t) {
  t.Skip("SELinux is present — skipping absence runtime test")
 }

 project := helpers.ProjectName(t)
 hostPath := t.TempDir()
 dir, _ := helpers.WriteCompose(t, composeWithBindMount(project, hostPath))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 containerName := fmt.Sprintf("%s-app", project)
 unitName := fmt.Sprintf("cq-%s-app.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // Without SELinux, no z label should appear on any mount
 helpers.AssertMountMissingOption(t, containerName, "/data", "z")
 helpers.AssertMountMissingOption(t, containerName, "/data-ro", "z")
 helpers.AssertMountMissingOption(t, containerName, "/data-rw", "z")
}

// --- Verbose output tests ---
// Verify the Cook stage logs the SELinux injection when -v is passed.

func TestSELinux_VerboseOutput_LogsInjection(t *testing.T) {
	helpers.SkipIfSELinuxNotEnabled(t)

 project := helpers.ProjectName(t)
 hostPath := t.TempDir()
 dir, _ := helpers.WriteCompose(t, composeWithBindMount(project, hostPath))

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run", "-v")

 // The Cook stage must log the SELinux z injection per the architecture
 if !strings.Contains(result.Stdout, ":z") {
  t.Fatalf("verbose dry-run missing SELinux injection log:\n%s", result.Stdout)
 }
}
