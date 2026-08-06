//go:build integration

package integration

import (
 "fmt"
 "os"
 "strconv"
 "strings"
 "testing"

 "github.com/Inoriol/comquad/tests/integration/helpers"
)

// skipIfRoot skips the test when running as UID 0.
// Rootless-specific tests must run as a non-root user.
func skipIfRoot(t *testing.T) {
 t.Helper()
 if os.Getuid() == 0 {
  t.Skip("rootless test must run as non-root user")
 }
}

// skipIfNotRoot skips the test when not running as UID 0.
func skipIfNotRoot(t *testing.T) {
 t.Helper()
 if os.Getuid() != 0 {
  t.Skip("this test must run as root")
 }
}

func TestRootless_TargetDirectory(t *testing.T) {
	skipIfRoot(t)
	helpers.SkipIfSystemdUnavailable(t)

	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 // Rootless target must be ~/.config/containers/systemd
 home, err := os.UserHomeDir()
 if err != nil {
  t.Fatalf("could not determine home dir: %v", err)
 }

 state := helpers.AssertProjectRegistered(t, project)
 for _, f := range state.Files {
  expected := fmt.Sprintf("%s/.config/containers/systemd", home)
  if !strings.HasPrefix(f, expected) {
   t.Fatalf("rootless file %q not in expected target dir %q", f, expected)
  }
 }
}

func TestRootless_PortOffset(t *testing.T) {
 skipIfRoot(t)

 project := helpers.ProjectName(t)

 // Use a privileged port — should be offset by ROOTLESS_PORT_OFFSET (2000)
 compose := `name: ` + project + `
services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "80:80"
`
 dir, _ := helpers.WriteCompose(t, compose)

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // Dry-run output must show the offset port (80 + 2000 = 2080)
 if !strings.Contains(result.Stdout, "2080") {
  t.Fatalf("expected port offset to 2080 in dry-run output, got:\n%s", result.Stdout)
 }

 // Must NOT contain the original privileged port as a published port
 if strings.Contains(result.Stdout, "PublishPort=80:80") {
  t.Fatalf("privileged port 80 was not offset in rootless mode:\n%s", result.Stdout)
 }
}

func TestRootless_PortOffset_NonPrivileged_Unchanged(t *testing.T) {
 skipIfRoot(t)

 project := helpers.ProjectName(t)

 // Non-privileged port — must NOT be offset
 compose := `name: ` + project + `
services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "8080:80"
`
 dir, _ := helpers.WriteCompose(t, compose)

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 if !strings.Contains(result.Stdout, "8080") {
  t.Fatalf("non-privileged port 8080 should be unchanged, got:\n%s", result.Stdout)
 }
}

func TestRootless_SystemdUserInstance(t *testing.T) {
	skipIfRoot(t)
	helpers.SkipIfSystemdUnavailable(t)

	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)

 // Must be active under the USER systemd instance, not system
 helpers.AssertUnitActive(t, unitName, true /* rootless */)

 // Must NOT appear in the system instance
 state := helpers.UnitActiveState(t, unitName, false /* system */)
 if state == "active" {
  t.Fatal("rootless unit must not be active in the system systemd instance")
 }
}

func TestRoot_TargetDirectory(t *testing.T) {
	skipIfNotRoot(t)
	helpers.SkipIfSystemdUnavailable(t)

	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 state := helpers.AssertProjectRegistered(t, project)
 for _, f := range state.Files {
  if !strings.HasPrefix(f, "/etc/containers/systemd") {
   t.Fatalf("root file %q not in expected target dir /etc/containers/systemd", f)
  }
 }
}

func TestRoot_PrivilegedPort_NoOffset(t *testing.T) {
 skipIfNotRoot(t)

 project := helpers.ProjectName(t)

 compose := `name: ` + project + `
services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "80:80"
`
 dir, _ := helpers.WriteCompose(t, compose)

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // Root mode: privileged ports must NOT be offset
 if !strings.Contains(result.Stdout, "PublishPort=80:80") {
  t.Fatalf("root mode should not offset privileged ports, got:\n%s", result.Stdout)
 }

 // Sanity: must not contain the offset value
 if strings.Contains(result.Stdout, strconv.Itoa(80+2000)) {
  t.Fatalf("root mode must not apply port offset, got:\n%s", result.Stdout)
 }
}
