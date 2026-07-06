//go:build integration

package integration

import (
 "fmt"
 "testing"
 "time"

 "tests/integration/helpers"
)

func TestLifecycle_StopStart(t *testing.T) {
 project := helpers.ProjectName(t)
 dir := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--project", project)
 })

 helpers.MustSucceed(t, dir, "up", "-d")

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 // stop
 helpers.MustSucceed(t, dir, "stop", "--project", project)
 helpers.AssertUnitInactive(t, unitName, false)

 // quadlet files must still exist — stop does not remove them
 if !helpers.UnitExists(t, unitName, false) {
  t.Fatal("unit file must still exist after stop")
 }

 // state entry must still exist
 helpers.AssertProjectRegistered(t, project)

 // start again
 helpers.MustSucceed(t, dir, "start", "--project", project)
 helpers.AssertUnitActive(t, unitName, false)
}

func TestLifecycle_Restart(t *testing.T) {
 project := helpers.ProjectName(t)
 dir := helpers.WriteCompose(t, helpers.SimpleCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--project", project)
 })

 helpers.MustSucceed(t, dir, "up", "-d")

 unitName := fmt.Sprintf("cq-%s-web.service", project)
 helpers.AssertUnitActive(t, unitName, false)

 helpers.MustSucceed(t, dir, "restart", "--project", project)

 // Give systemd time to cycle the unit
 time.Sleep(2 * time.Second)
 helpers.AssertUnitActive(t, unitName, false)
}

func TestLifecycle_StopSpecificService(t *testing.T) {
 project := helpers.ProjectName(t)
 dir := helpers.WriteCompose(t, helpers.MultiServiceCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--project", project)
 })

 helpers.MustSucceed(t, dir, "up", "-d")

 webUnit := fmt.Sprintf("cq-%s-web.service", project)
 apiUnit := fmt.Sprintf("cq-%s-api.service", project)

 helpers.AssertUnitActive(t, webUnit, false)
 helpers.AssertUnitActive(t, apiUnit, false)

 // Stop only the web service
 helpers.MustSucceed(t, dir, "stop", "--project", project, "web")
 helpers.AssertUnitInactive(t, webUnit, false)

 // api must still be running
 helpers.AssertUnitActive(t, apiUnit, false)
}

func TestLifecycle_StartSpecificService(t *testing.T) {
 project := helpers.ProjectName(t)
 dir := helpers.WriteCompose(t, helpers.MultiServiceCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--project", project)
 })

 helpers.MustSucceed(t, dir, "up", "-d")

 webUnit := fmt.Sprintf("cq-%s-web.service", project)
 apiUnit := fmt.Sprintf("cq-%s-api.service", project)

 // Stop both
 helpers.MustSucceed(t, dir, "stop", "--project", project)
 helpers.AssertUnitInactive(t, webUnit, false)
 helpers.AssertUnitInactive(t, apiUnit, false)

 // Start only api
 helpers.MustSucceed(t, dir, "start", "--project", project, "api")
 helpers.AssertUnitActive(t, apiUnit, false)

 // web must still be inactive
 if helpers.UnitActiveState(t, webUnit, false) == "active" {
  t.Fatal("web unit should still be inactive after starting only api")
 }
}

func TestLifecycle_RestartSpecificService(t *testing.T) {
 project := helpers.ProjectName(t)
 dir := helpers.WriteCompose(t, helpers.MultiServiceCompose(project))

 t.Cleanup(func() {
  helpers.Comquad(t, dir, "down", "--project", project)
 })

 helpers.MustSucceed(t, dir, "up", "-d")

 webUnit := fmt.Sprintf("cq-%s-web.service", project)
 apiUnit := fmt.Sprintf("cq-%s-api.service", project)

 helpers.AssertUnitActive(t, webUnit, false)
 helpers.AssertUnitActive(t, apiUnit, false)

 // Restart only web
 helpers.MustSucceed(t, dir, "restart", "--project", project, "web")
 time.Sleep(2 * time.Second)

 helpers.AssertUnitActive(t, webUnit, false)
 // api should be untouched
 helpers.AssertUnitActive(t, apiUnit, false)
}

func TestLifecycle_StopRequiresExistingProject(t *testing.T) {
 // Stopping a project that was never deployed should fail cleanly
 helpers.MustFail(t, "", "stop", "--project", "cqt-nonexistent-project")
}

func TestLifecycle_StartRequiresExistingProject(t *testing.T) {
 helpers.MustFail(t, "", "start", "--project", "cqt-nonexistent-project")
}

func TestLifecycle_RestartRequiresExistingProject(t *testing.T) {
 helpers.MustFail(t, "", "restart", "--project", "cqt-nonexistent-project")
}
