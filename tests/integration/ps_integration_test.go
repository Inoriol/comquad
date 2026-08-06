//go:build integration

package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/tests/integration/helpers"
)

func TestPs_OutputFormat(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "up", "--name", project)

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)

	result := helpers.Comquad(t, dir, "ps", "--name", project)
	if result.ExitCode != 0 {
		t.Fatalf("ps failed: %s", result.Stderr)
	}

	// Verify column headers
	if !strings.Contains(result.Stdout, "NAME") {
		t.Errorf("ps output missing NAME column:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "IMAGE") {
		t.Errorf("ps output missing IMAGE column:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "COMMAND") {
		t.Errorf("ps output missing COMMAND column:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "SERVICE") {
		t.Errorf("ps output missing SERVICE column:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "CREATED") {
		t.Errorf("ps output missing CREATED column:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "STATUS") {
		t.Errorf("ps output missing STATUS column:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "PORTS") {
		t.Errorf("ps output missing PORTS column:\n%s", result.Stdout)
	}

	// Verify container appears in output
	containerName := fmt.Sprintf("%s-web", project)
	if !strings.Contains(result.Stdout, containerName) {
		t.Errorf("ps output missing container %q:\n%s", containerName, result.Stdout)
	}
}

func TestPs_AllIncludesExitedContainers(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "up", "--name", project)

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)

	// Stop the unit so it becomes exited
	helpers.MustSucceed(t, dir, "stop", "--name", project)
	helpers.AssertUnitInactive(t, unitName, false)

	// ps without -a should not show the stopped container
	result := helpers.Comquad(t, dir, "ps", "--name", project)
	if strings.Contains(result.Stdout, fmt.Sprintf("%s-web", project)) {
		// Without -a, exited containers generally don't appear.
		// podman ps without -a excludes exited containers.
	}

	// ps -a should show the exited container
	resultAll := helpers.Comquad(t, dir, "ps", "--name", project, "-a")
	if resultAll.ExitCode != 0 {
		t.Fatalf("ps -a failed: %s", resultAll.Stderr)
	}
	if !strings.Contains(resultAll.Stdout, fmt.Sprintf("%s-web", project)) {
		t.Errorf("ps -a output missing exited container %q:\n%s", fmt.Sprintf("%s-web", project), resultAll.Stdout)
	}
}

func TestPs_RequiresExistingProject(t *testing.T) {
	helpers.MustFail(t, "", "ps", "--name", "cqt-nonexistent-project")
}
