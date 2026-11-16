package main

import (
	"os"
	"testing"
)

func TestIsProviderMode_DockerCompose(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"docker-compose", "up", "-d"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for docker-compose")
	}
}

func TestIsProviderMode_PodmanCompose(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"podman-compose", "up", "-d"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for podman-compose")
	}
}

func TestIsProviderMode_DockerComposeWithFlags(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"docker-compose", "-f", "compose.yaml", "up", "-d"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for docker-compose with flags")
	}
}

func TestIsProviderMode_DockerComposeNoArgs(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"docker-compose"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for docker-compose with no args")
	}
}

func TestIsProviderMode_ComquadComposeCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "compose", "up", "-d"}
	if isProviderMode() {
		t.Error("expected isProviderMode() to return false for comquad compose (explicit subcommand)")
	}
}

func TestIsProviderMode_ComquadUpCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "up"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad up (provider mode)")
	}
}

func TestIsProviderMode_ComquadDownCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "down"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad down (provider mode)")
	}
}

func TestIsProviderMode_ComquadPsCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "ps"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad ps (provider mode)")
	}
}

func TestIsProviderMode_ComquadLogsCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "logs"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad logs (provider mode)")
	}
}

func TestIsProviderMode_ComquadStartCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "start"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad start (provider mode)")
	}
}

func TestIsProviderMode_ComquadStopCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "stop"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad stop (provider mode)")
	}
}

func TestIsProviderMode_ComquadRestartCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "restart"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad restart (provider mode)")
	}
}

func TestIsProviderMode_ComquadExecCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "exec", "web", "ls"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad exec (provider mode)")
	}
}

func TestIsProviderMode_ComquadBuildCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "build"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad build (provider mode)")
	}
}

func TestIsProviderMode_ComquadPullCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "pull"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad pull (provider mode)")
	}
}

func TestIsProviderMode_ComquadListCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "list"}
	if isProviderMode() {
		t.Error("expected isProviderMode() to return false for comquad list (not a compose command)")
	}
}

func TestIsProviderMode_ComquadCheckCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "check"}
	if isProviderMode() {
		t.Error("expected isProviderMode() to return false for comquad check (not a compose command)")
	}
}

func TestIsProviderMode_ComquadNoArgs(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad"}
	if isProviderMode() {
		t.Error("expected isProviderMode() to return false for comquad with no args")
	}
}

func TestIsProviderMode_ComquadHelpFlag(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "--help"}
	if isProviderMode() {
		t.Error("expected isProviderMode() to return false for comquad --help")
	}
}

func TestIsProviderMode_ComquadWithFlagsBeforeCommand(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "-f", "compose.yaml", "up", "-d"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad with flags before compose command")
	}
}

func TestIsProviderMode_ComquadWithProjectName(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "-p", "myproject", "up", "-d"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for comquad with -p flag before compose command")
	}
}

func TestIsProviderMode_DockerComposeWithPath(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"/usr/local/bin/docker-compose", "up", "-d"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for docker-compose with full path")
	}
}

func TestIsProviderMode_PodmanComposeWithPath(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"/home/user/.local/bin/podman-compose", "ps"}
	if !isProviderMode() {
		t.Error("expected isProviderMode() to return true for podman-compose with full path")
	}
}

func TestIsProviderMode_ComquadComposeDown(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "compose", "down"}
	if isProviderMode() {
		t.Error("expected isProviderMode() to return false for comquad compose down")
	}
}

func TestIsProviderMode_ComquadComposePs(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"comquad", "compose", "ps"}
	if isProviderMode() {
		t.Error("expected isProviderMode() to return false for comquad compose ps")
	}
}
