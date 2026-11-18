//go:build integration

package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/tests/integration/helpers"
)

func TestPull_SimpleService(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Pull images without starting containers
	output := helpers.MustSucceed(t, dir, "pull", "--name", project)

	if !strings.Contains(output.Stdout, "Successfully pulled images") {
		t.Errorf("expected success message, got: %s", output.Stdout)
	}

	// Verify images were pulled but containers are not running
	imageName := fmt.Sprintf("%s-web:latest", project)
	if !helpers.ImageExists(t, imageName) {
		t.Errorf("expected image %q to exist after pull", imageName)
	}

	// Container should not be running
	containerName := fmt.Sprintf("%s-web", project)
	if helpers.ContainerRunning(t, containerName) {
		t.Error("container should not be running after pull")
	}
}

func TestPull_DryRun(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Dry run should not pull anything
	output := helpers.MustSucceed(t, dir, "pull", "--name", project, "--dry-run")

	if !strings.Contains(output.Stdout, "Dry run") {
		t.Errorf("expected dry run output, got: %s", output.Stdout)
	}

	// Image should not exist after dry run
	imageName := fmt.Sprintf("%s-web:latest", project)
	if helpers.ImageExists(t, imageName) {
		t.Errorf("image %q should not exist after dry run", imageName)
	}
}

func TestCompose_Up_Detach(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "compose", "down", "--name", project)
	})

	// compose up -d should detach
	helpers.MustSucceed(t, dir, "compose", "up", "-d", "--name", project)

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)
}

func TestCompose_Down_Volumes(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.WithVolumeCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "compose", "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "compose", "up", "-d", "--name", project)

	volumeName := fmt.Sprintf("cq-%s-dbdata", project)
	if !helpers.VolumeExists(t, volumeName) {
		t.Fatalf("expected volume %q to exist after up", volumeName)
	}

	// compose down --volumes should remove volumes
	helpers.MustSucceed(t, dir, "compose", "down", "--name", project, "--volumes")

	if helpers.VolumeExists(t, volumeName) {
		t.Error("volume should be removed after compose down --volumes")
	}
}

func TestCompose_Ps(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "compose", "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "compose", "up", "-d", "--name", project)

	// compose ps should show running containers
	output := helpers.MustSucceed(t, dir, "compose", "ps", "--name", project)

	if !strings.Contains(output.Stdout, "web") {
		t.Errorf("expected 'web' service in ps output, got: %s", output.Stdout)
	}
}

func TestCompose_Logs(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "compose", "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "compose", "up", "-d", "--name", project)

	// compose logs should show container logs
	output := helpers.MustSucceed(t, dir, "compose", "logs", "--name", project, "--tail", "5")

	// Just verify it doesn't error - actual log content depends on the container
	if output.Stdout == "" {
		t.Error("expected some log output")
	}
}

func TestCompose_Exec(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "compose", "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "compose", "up", "-d", "--name", project)

	// Wait for container to be ready
	helpers.WaitForContainer(t, fmt.Sprintf("%s-web", project))

	// compose exec should run command in container
	output := helpers.MustSucceed(t, dir, "compose", "exec", "--name", project, "web", "echo", "hello")

	if !strings.Contains(output.Stdout, "hello") {
		t.Errorf("expected 'hello' in exec output, got: %s", output.Stdout)
	}
}

func TestCompose_Lifecycle(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "compose", "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "compose", "up", "-d", "--name", project)

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)

	// Stop
	helpers.MustSucceed(t, dir, "compose", "stop", "--name", project, "web")
	helpers.AssertUnitInactive(t, unitName, false)

	// Start
	helpers.MustSucceed(t, dir, "compose", "start", "--name", project, "web")
	helpers.AssertUnitActive(t, unitName, false)

	// Restart
	helpers.MustSucceed(t, dir, "compose", "restart", "--name", project, "web")
	helpers.AssertUnitActive(t, unitName, false)
}
