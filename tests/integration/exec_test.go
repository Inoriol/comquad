//go:build integration

package integration

import (
 "fmt"
 "strings"
 "testing"

 "comquad/tests/integration/helpers"
)

func TestExec_RunsCommandInContainer(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // Run a simple command inside the container
 result := helpers.MustSucceed(t, dir, "exec",
  "--name", project,
  "--tty=false",
  "web",
  "--", "echo", "hello-from-container",
 )

 if !strings.Contains(result.Stdout, "hello-from-container") {
  t.Fatalf("exec output missing expected string, got:\n%s", result.Stdout)
 }
}

func TestExec_WithUser(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // Run whoami as root inside the container
 result := helpers.MustSucceed(t, dir, "exec",
  "--name", project,
  "--tty=false",
  "--user", "root",
  "web",
  "--", "whoami",
 )

 if !strings.Contains(result.Stdout, "root") {
  t.Fatalf("expected whoami to return root, got:\n%s", result.Stdout)
 }
}

func TestExec_NonexistentService_Errors(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 helpers.MustFail(t, dir, "exec",
  "--name", project,
  "--tty=false",
  "nonexistent-service",
  "--", "echo", "hi",
 )
}

func TestExec_RequiresRunningContainer(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)
 helpers.MustSucceed(t, dir, "stop", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitInactive(t, unitName, false)

 // exec into a stopped container should fail
 helpers.MustFail(t, dir, "exec",
  "--name", project,
  "--tty=false",
  "web",
  "--", "echo", "hi",
 )
}
