//go:build integration

package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/tests/integration/helpers"
)

func TestJobs_UpDoesNotStartJobs(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.JobCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project, "-y")
	})

	helpers.MustSucceed(t, dir, "up", "--name", project, "--no-diff")

	webUnit := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, webUnit, false)

	jobUnit := fmt.Sprintf("cq-%s-migrate.service", project)
	state := helpers.UnitActiveState(t, jobUnit, false)
	if state == "active" {
		t.Fatal("job container should not be started during up")
	}
}

func TestJobs_RunExecutesJob(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.JobCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project, "-y")
	})

	helpers.MustSucceed(t, dir, "up", "--name", project, "--no-diff")

	result := helpers.MustSucceed(t, dir, "run", "--name", project, "migrate")
	if result.ExitCode != 0 {
		t.Fatalf("run job failed: %s", result.Stderr)
	}
}

func TestJobs_RunNonExistentJobFails(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.JobCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project, "-y")
	})

	helpers.MustSucceed(t, dir, "up", "--name", project, "--no-diff")

	result := helpers.Comquad(t, dir, "run", "--name", project, "nonexistent")
	if result.ExitCode == 0 {
		t.Fatal("expected run of nonexistent job to fail")
	}
}

func TestJobs_DownStopsTimers(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.JobScheduledCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project, "-y")
	})

	helpers.MustSucceed(t, dir, "up", "--name", project, "--no-diff")

	timerUnit := fmt.Sprintf("cq-%s-cleanup.timer", project)
	helpers.AssertUnitActive(t, timerUnit, false)

	helpers.MustSucceed(t, dir, "down", "--name", project, "-y")

	state := helpers.UnitActiveState(t, timerUnit, false)
	if state == "active" {
		t.Fatal("timer should be stopped after down")
	}
}

func TestJobs_PsShowsJobType(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.JobCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project, "-y")
	})

	helpers.MustSucceed(t, dir, "up", "--name", project, "--no-diff")

	result := helpers.MustSucceed(t, dir, "ps", "--name", project, "--json")
	if !strings.Contains(result.Stdout, `"type":"service"`) {
		t.Fatal("ps output should contain service type")
	}
}

func TestJobs_ViewShowsJobs(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.JobCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project, "-y")
	})

	helpers.MustSucceed(t, dir, "up", "--name", project, "--no-diff")

	result := helpers.MustSucceed(t, dir, "view", "--name", project, "--json")
	if !strings.Contains(result.Stdout, `"jobs"`) {
		t.Fatal("view output should contain jobs section")
	}
	if !strings.Contains(result.Stdout, `"migrate"`) {
		t.Fatal("view output should contain job name 'migrate'")
	}
}

func TestJobs_ViewShowsTimers(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.JobScheduledCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project, "-y")
	})

	helpers.MustSucceed(t, dir, "up", "--name", project, "--no-diff")

	result := helpers.MustSucceed(t, dir, "view", "--name", project, "--json")
	if !strings.Contains(result.Stdout, `"timer"`) {
		t.Fatal("view output should contain timer resource")
	}
}

func TestJobs_ListShowsJobCount(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.JobScheduledCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project, "-y")
	})

	helpers.MustSucceed(t, dir, "up", "--name", project, "--no-diff")

	result := helpers.MustSucceed(t, dir, "list", "--json")
	if !strings.Contains(result.Stdout, `"jobs"`) {
		t.Fatal("list output should contain jobs field")
	}
	if !strings.Contains(result.Stdout, `"timers"`) {
		t.Fatal("list output should contain timers in resources")
	}
}

func TestJobs_ComposeRun(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.JobCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "compose", "down", "--name", project, "-y")
	})

	helpers.MustSucceed(t, dir, "compose", "up", "-d", "--name", project, "--no-diff")

	result := helpers.MustSucceed(t, dir, "compose", "run", "--name", project, "migrate")
	if result.ExitCode != 0 {
		t.Fatalf("compose run job failed: %s", result.Stderr)
	}
}
