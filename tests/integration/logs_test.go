//go:build integration

package integration

import (
 "fmt"
 "strings"
 "testing"
 "time"

 "comquad/tests/integration/helpers"
)

func TestLogs_RunningUnit(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // Give the container a moment to emit startup logs
 time.Sleep(3 * time.Second)

 result := helpers.MustSucceed(t, dir, "logs", "--name", project)

 // Logs command must succeed and return some output for a running unit
 if result.Stdout == "" && result.Stderr == "" {
  t.Fatal("expected some log output for running unit, got nothing")
 }
}

func TestLogs_TailFlag(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)
 time.Sleep(3 * time.Second)

 result := helpers.MustSucceed(t, dir, "logs", "--name", project, "--tail", "5")

 lines := nonEmptyLines(result.Stdout)
 if len(lines) > 5 {
  t.Fatalf("--tail 5 returned %d lines:\n%s", len(lines), result.Stdout)
 }
}

func TestLogs_StoppedUnit(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)
 time.Sleep(2 * time.Second)

 // Stop the unit — logs should still be retrievable from history
 helpers.MustSucceed(t, dir, "stop", "--name", project)
 helpers.AssertUnitInactive(t, unitName, false)

 result := helpers.MustSucceed(t, dir, "logs", "--name", project)
 // Should not error even for a stopped unit
 _ = result
}

func TestLogs_SpecificService(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.MultiServiceCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 for _, svc := range []string{"web", "api"} {
  unit := fmt.Sprintf("cq-%s-%s.service", project, svc)
  helpers.AssertUnitActive(t, unit, false)
 }
 time.Sleep(3 * time.Second)

 // Request logs for web only
 result := helpers.MustSucceed(t, dir, "logs", "--name", project, "web")
 _ = result
}

func TestLogs_MultiService_LinesPrefixed(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.MultiServiceCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 for _, svc := range []string{"web", "api"} {
  unit := fmt.Sprintf("cq-%s-%s.service", project, svc)
  helpers.AssertUnitActive(t, unit, false)
 }
 time.Sleep(3 * time.Second)

 result := helpers.MustSucceed(t, dir, "logs", "--name", project)

 // When querying multiple units each non-separator line must be prefixed
 // with [<unit-name>]
 lines := nonEmptyLines(result.Stdout)
 for _, line := range lines {
  if strings.HasPrefix(line, "--") {
   // journalctl separator — pass through unmodified, skip check
   continue
  }
  if !strings.HasPrefix(line, "[") {
   t.Fatalf("log line missing unit prefix: %q", line)
  }
 }
}

func TestLogs_NoEmptyLines(t *testing.T) {
 project := helpers.ProjectName(t)
 dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--name", project)
 })

 helpers.MustSucceed(t, dir, "up", "--name", project)

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)
 time.Sleep(3 * time.Second)

 result := helpers.MustSucceed(t, dir, "logs", "--name", project)

 // Per architecture: all empty lines are stripped in every code path
 for i, line := range strings.Split(result.Stdout, "\n") {
  if line == "" && i < len(strings.Split(result.Stdout, "\n"))-1 {
   t.Fatalf("found empty line at position %d in log output", i)
  }
 }
}

func TestLogs_RequiresExistingProject(t *testing.T) {
 helpers.MustFail(t, "", "logs", "--name", "cqt-nonexistent-project")
}

// nonEmptyLines splits output into lines, filtering out empty ones.
func nonEmptyLines(s string) []string {
 var result []string
 for _, line := range strings.Split(s, "\n") {
  if strings.TrimSpace(line) != "" {
   result = append(result, line)
  }
 }
 return result
}
