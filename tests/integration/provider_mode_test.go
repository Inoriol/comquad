//go:build integration

package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/tests/integration/helpers"
)

func TestProviderMode_DockerCompose_Up_Down(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	if !strings.Contains(result.Stdout, "Successfully deployed project") {
		t.Fatalf("expected successful deployment message, got:\n%s", result.Stdout)
	}

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)
	helpers.AssertContainerRunning(t, fmt.Sprintf("%s-web", project))

	result = helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")

	if !strings.Contains(result.Stdout, "Successfully removed project") {
		t.Fatalf("expected successful removal message, got:\n%s", result.Stdout)
	}

	helpers.AssertUnitInactive(t, unitName, false)
	helpers.AssertContainerGone(t, fmt.Sprintf("%s-web", project))
}

func TestProviderMode_DockerCompose_Ps(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "ps")

	if !strings.Contains(result.Stdout, fmt.Sprintf("%s-web", project)) {
		t.Fatalf("expected container name in ps output, got:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "running") || !strings.Contains(result.Stdout, "Up") {
		t.Fatalf("expected running status in ps output, got:\n%s", result.Stdout)
	}
}

func TestProviderMode_DockerCompose_Logs(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "logs", "--tail", "5")

	if result.Stdout == "" {
		t.Fatal("expected some log output")
	}
}

func TestProviderMode_DockerCompose_Stop_Start(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)

	helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "stop")
	helpers.AssertUnitInactive(t, unitName, false)

	helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "start")
	helpers.AssertUnitActive(t, unitName, false)
}

func TestProviderMode_DockerCompose_Restart(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "restart")

	if !strings.Contains(result.Stdout, "Successfully restarted project") {
		t.Fatalf("expected restart success message, got:\n%s", result.Stdout)
	}

	helpers.AssertUnitActive(t, unitName, false)
}

func TestProviderMode_DockerCompose_Exec(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "exec", "web", "cat", "/etc/os-release")

	if !strings.Contains(result.Stdout, "Alpine") {
		t.Fatalf("expected Alpine in exec output, got:\n%s", result.Stdout)
	}
}

func TestProviderMode_DockerCompose_DryRun(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d", "--dry-run")

	if !strings.Contains(result.Stdout, "Dry run") {
		t.Fatalf("expected dry run message, got:\n%s", result.Stdout)
	}

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitInactive(t, unitName, false)
}

func TestProviderMode_PodmanCompose_Up_Down(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "podman-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	result := helpers.ProviderBinaryMustSucceed(t, dir, "podman-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	if !strings.Contains(result.Stdout, "Successfully deployed project") {
		t.Fatalf("expected successful deployment message, got:\n%s", result.Stdout)
	}

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)
	helpers.AssertContainerRunning(t, fmt.Sprintf("%s-web", project))

	helpers.ProviderBinaryMustSucceed(t, dir, "podman-compose", "-f", dir+"/compose.yaml", "-p", project, "down")

	helpers.AssertUnitInactive(t, unitName, false)
	helpers.AssertContainerGone(t, fmt.Sprintf("%s-web", project))
}

func TestProviderMode_DockerCompose_Help(t *testing.T) {
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "--help")

	if !strings.Contains(result.Stdout, "Docker Compose compatible") {
		t.Fatalf("expected compose help text, got:\n%s", result.Stdout)
	}
}

func TestProviderMode_DockerCompose_UpHelp(t *testing.T) {
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "up", "--help")

	if !strings.Contains(result.Stdout, "Builds, (re)creates, starts") {
		t.Fatalf("expected up help text, got:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "--detach") {
		t.Fatalf("expected --detach flag in help, got:\n%s", result.Stdout)
	}
}

func TestProviderMode_DockerCompose_MultiService(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.MultiServiceCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	for _, svc := range []string{"web", "api"} {
		unit := fmt.Sprintf("cq-%s-%s.service", project, svc)
		helpers.AssertUnitActive(t, unit, false)
		helpers.AssertContainerRunning(t, fmt.Sprintf("%s-%s", project, svc))
	}

	helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")

	for _, svc := range []string{"web", "api"} {
		unit := fmt.Sprintf("cq-%s-%s.service", project, svc)
		helpers.AssertUnitInactive(t, unit, false)
	}
}

func TestProviderMode_DockerCompose_FileFlag(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	if !strings.Contains(result.Stdout, "Successfully deployed project") {
		t.Fatalf("expected successful deployment with -f flag, got:\n%s", result.Stdout)
	}
}

func TestProviderMode_DockerCompose_ProjectNameFlag(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d")

	if !strings.Contains(result.Stdout, fmt.Sprintf("project: %s", project)) {
		t.Fatalf("expected project name in output, got:\n%s", result.Stdout)
	}
}

func TestProviderMode_DockerCompose_NoDiff(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "up", "-d", "--no-diff")

	if !strings.Contains(result.Stdout, "Successfully deployed project") {
		t.Fatalf("expected successful deployment with --no-diff, got:\n%s", result.Stdout)
	}
}

func TestProviderMode_DockerCompose_Pull(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "pull")

	if !strings.Contains(result.Stdout, "Successfully pulled") {
		t.Fatalf("expected pull success message, got:\n%s", result.Stdout)
	}
}

func TestProviderMode_DockerCompose_Build(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)

	compose := `name: ` + project + `
services:
  web:
    build: .
    ports:
      - "8080:80"
`
	dir, _ := helpers.WriteCompose(t, compose)
	helpers.WriteFile(t, dir, "Dockerfile", "FROM docker.io/library/alpine:latest\nCMD [\"sleep\", \"infinity\"]\n")

	t.Cleanup(func() {
		helpers.ProviderBinary(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "down")
	})

	result := helpers.ProviderBinaryMustSucceed(t, dir, "docker-compose", "-f", dir+"/compose.yaml", "-p", project, "build")

	if !strings.Contains(result.Stdout, "Successfully built") {
		t.Fatalf("expected build success message, got:\n%s", result.Stdout)
	}
}
