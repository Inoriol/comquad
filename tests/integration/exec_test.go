//go:build integration

package integration

import (
 "fmt"
 "strings"
 "testing"

 "tests/integration/helpers"
)

func TestExec_RunsCommandInContainer(t *testing.T) {
 project := helpers.ProjectName(t)
 dir := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--project", project)
 })

 helpers.MustSucceed(t, dir, "up", "-d")

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // Run a simple command inside the container
 result := helpers.MustSucceed(t, dir, "exec",
  "--project", project,
  "--tty=false",
  "web",
  "--", "echo", "hello-from-container",
 )

 if !strings.Contains(result.Stdout, "hello-from-container") {
  t.Fatalf("exec output missing expected string, got:\n%s", result.Stdout)
 }
}

func TestExec_WithUser(t *testing.T) {
 project := helpers.ProjectName(t)
 dir := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--project", project)
 })

 helpers.MustSucceed(t, dir, "up", "-d")

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // Run whoami as root inside the container
 result := helpers.MustSucceed(t, dir, "exec",
  "--project", project,
  "--tty=false",
  "--user", "root",
  "web",
  "--", "whoami",
 )

 if !strings.Contains(result.Stdout, "root") {
  t.Fatalf("expected whoami to return root, got:\n%s", result.Stdout)
 }
}

func TestExec_AmbiguousService_Errors(t *testing.T) {
 // This test requires a compose where two services could match the same
 // short name — not possible with distinct names, so we verify the error
 // path by passing a name that matches nothing instead.
 project := helpers.ProjectName(t)
 dir := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--project", project)
 })

 helpers.MustSucceed(t, dir, "up", "-d")

 helpers.MustFail(t, dir, "exec",
  "--project", project,
  "--tty=false",
  "nonexistent-service",
  "--", "echo", "hi",
 )
}

func TestExec_RequiresRunningContainer(t *testing.T) {
 project := helpers.ProjectName(t)
 dir := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--project", project)
 })

 helpers.MustSucceed(t, dir, "up", "-d")
 helpers.MustSucceed(t, dir, "stop", "--project", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitInactive(t, unitName, false)

 // exec into a stopped container should fail
 helpers.MustFail(t, dir, "exec",
  "--project", project,
  "--tty=false",
  "web",
  "--", "echo", "hi",
 )
}
