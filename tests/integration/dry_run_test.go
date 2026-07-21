//go:build integration

package integration

import (
 "fmt"
 "strings"
 "testing"

 "comquad/tests/integration/helpers"
)

func TestDryRun_NoFilesWritten(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // Output must mention the would-be target path
 if !strings.Contains(result.Stdout, fmt.Sprintf("cq-%s", project)) {
  t.Fatalf("dry-run output missing project prefix, got:\n%s", result.Stdout)
 }

 // No unit files should exist in systemd target dir
 unitName := fmt.Sprintf("cq-%s-web.container", project)
 if helpers.UnitExists(t, unitName, false) {
  t.Fatal("dry-run must not create systemd unit files")
 }

 // No state entry should be registered
 projects := helpers.ReadStateFile(t)
 if _, ok := projects[project]; ok {
  t.Fatal("dry-run must not register project in state file")
 }
}

func TestDryRun_ShowsFileContents(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // Dry-run should print quadlet file contents
 for _, expected := range []string{
  "[Container]",
  "[Install]",
  fmt.Sprintf("cq-%s-web.container", project),
 } {
  if !strings.Contains(result.Stdout, expected) {
   t.Fatalf("dry-run output missing %q:\n%s", expected, result.Stdout)
  }
 }
}

func TestDryRun_ShowsImageActions(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // Should mention image pull or build action for the web service
 if !strings.Contains(result.Stdout, "nginx") {
  t.Fatalf("dry-run output missing image reference, got:\n%s", result.Stdout)
 }
}

func TestDryRun_MultiService_NoSideEffects(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.MultiServiceCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 // Neither service unit should exist
 for _, svc := range []string{"web", "api"} {
  unit := fmt.Sprintf("cq-%s-%s.container", project, svc)
  if helpers.UnitExists(t, unit, false) {
   t.Fatalf("dry-run must not create unit file for service %q", svc)
  }
 }

 // State must be clean
 projects := helpers.ReadStateFile(t)
 if _, ok := projects[project]; ok {
  t.Fatal("dry-run must not register multi-service project in state file")
 }
}

func TestDryRun_WithVolume_NoSideEffects(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.WithVolumeCompose(project))

 helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

 volumeName := fmt.Sprintf("cq-%s-dbdata", project)
 if helpers.VolumeExists(t, volumeName) {
  t.Fatalf("dry-run must not create Podman volume %q", volumeName)
 }
}
