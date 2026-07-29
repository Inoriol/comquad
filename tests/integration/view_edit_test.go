//go:build integration

package integration

import (
 "fmt"
 "strings"
 "testing"

 "comquad/tests/integration/helpers"
)

func TestView_ProjectSummary(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 result := helpers.MustSucceed(t, dir, "view", "--name", project)

 // Project view must show unit name, active state, and healthy status
 for _, expected := range []string{
  unitName,
  "active",
  "healthy",
 } {
  if !strings.Contains(result.Stdout, expected) {
   t.Fatalf("view output missing %q:\n%s", expected, result.Stdout)
  }
 }
}

func TestView_ProjectStatus_Degraded(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.MultiServiceCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 webUnit := fmt.Sprintf("cq-%s-web.service", project)
 apiUnit := fmt.Sprintf("cq-%s-api.service", project)
 helpers.AssertUnitActive(t, webUnit, false)
 helpers.AssertUnitActive(t, apiUnit, false)

 // Stop only one service to trigger degraded state
 helpers.MustSucceed(t, dir, "stop", "--name", project, "web")
 helpers.AssertUnitInactive(t, webUnit, false)

 result := helpers.MustSucceed(t, dir, "view", "--name", project)

 if !strings.Contains(result.Stdout, "degraded") {
  t.Fatalf("expected degraded status when only some units active:\n%s", result.Stdout)
 }
}

func TestView_ProjectStatus_Down(t *testing.T) {
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

 result := helpers.MustSucceed(t, dir, "view", "--name", project)

 if !strings.Contains(result.Stdout, "down") {
  t.Fatalf("expected down status when no units active:\n%s", result.Stdout)
 }
}

// TestView_UnitFile_MatchingPatterns exercises all five resolution patterns
// described in the architecture: short name, full name without extension,
// with .service suffix, with .container suffix, and internal podman name.
func TestView_UnitFile_MatchingPatterns(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 patterns := []string{
  "web",                                      // short name
  fmt.Sprintf("cq-%s-web", project),          // full name without extension
  fmt.Sprintf("cq-%s-web.service", project),  // with .service suffix
  fmt.Sprintf("cq-%s-web.container", project), // with .container suffix
  fmt.Sprintf("%s-web", project),              // internal podman name
 }

 for _, pattern := range patterns {
  t.Run(pattern, func(t *testing.T) {
   result := helpers.MustSucceed(t, dir, "view", "--name", project, pattern)
   // Unit file view must print the quadlet file contents
   if !strings.Contains(result.Stdout, "[Container]") {
    t.Fatalf("view with pattern %q missing [Container] section:\n%s",
     pattern, result.Stdout)
   }
  })
 }
}

func TestView_RequiresExistingProject(t *testing.T) {
 helpers.MustFail(t, "", "view", "--name", "cqt-nonexistent-project")
}

func TestEdit_NoReload_OpensWithoutRestart(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // Use EDITOR=true so the editor exits immediately without modifying files.
 // --no-reload ensures no daemon-reload or restart is triggered.
 result := helpers.Comquad(t, dir, "edit",
  "--name", project,
  "--no-reload",
 )

 // Should succeed — true always exits 0
 if result.ExitCode != 0 {
  t.Fatalf("edit --no-reload failed: %s", result.Stderr)
 }

 // Unit must still be active — no reload was triggered
 helpers.AssertUnitActive(t, unitName, false)
}
