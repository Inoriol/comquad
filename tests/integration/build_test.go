//go:build integration

package integration

import (
	"fmt"
	"testing"

	"comquad/tests/integration/helpers"
)

func TestUpDown_WithBuildBlocks(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteBuildCompose(t, project)

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Build and deploy
	helpers.MustSucceed(t, dir, "up", "--name", project)

	// The container should be running
	containerName := fmt.Sprintf("%s-web", project)
	helpers.AssertContainerRunning(t, containerName)

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)

	// Tear down and verify
	helpers.MustSucceed(t, dir, "down", "--name", project)
	helpers.AssertUnitInactive(t, unitName, false)
	helpers.AssertContainerGone(t, containerName)
}
