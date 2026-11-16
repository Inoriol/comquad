//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/tests/integration/helpers"
)

func skipIfNotRootful(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("AutoNS tests require rootful mode")
	}
}

func skipIfNoContainersSubuid(t *testing.T) {
	data, err := os.ReadFile("/etc/subuid")
	if err != nil {
		t.Skipf("cannot read /etc/subuid: %v", err)
	}
	if !strings.Contains(string(data), "containers:") {
		t.Skip("/etc/subuid does not contain 'containers:' entry required for AutoNS")
	}
}

func TestAutoNS_Deployment(t *testing.T) {
	skipIfNotRootful(t)
	skipIfNoContainersSubuid(t)
	helpers.SkipIfSystemdUnavailable(t)

	project := helpers.ProjectName(t)
	compose := fmt.Sprintf(`
services:
  db:
    image: docker.io/library/alpine:latest
    command: sleep infinity
  web:
    image: docker.io/library/alpine:latest
    command: sleep infinity
    depends_on:
      - db
  cache:
    image: docker.io/library/alpine:latest
    command: sleep infinity
    depends_on:
      - db
`)
	dir, _ := helpers.WriteCompose(t, compose)

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Deploy with AutoNS enabled
	result := helpers.ComquadWithEnv(t, dir, map[string]string{"COMQUAD_AUTONS": "1"}, "up", "--name", project, "--no-diff")
	if result.ExitCode != 0 {
		t.Fatalf("comquad up with AutoNS failed:\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
	}

	// Verify all containers are running
	for _, svc := range []string{"db", "web", "cache"} {
		containerName := fmt.Sprintf("%s-%s", project, svc)
		helpers.AssertContainerRunning(t, containerName)
	}

	// Verify UID mappings - all containers should share the same namespace
	dbPID := getContainerPID(t, fmt.Sprintf("%s-db", project))
	webPID := getContainerPID(t, fmt.Sprintf("%s-web", project))
	cachePID := getContainerPID(t, fmt.Sprintf("%s-cache", project))

	dbUIDMap := readUIDMap(t, dbPID)
	webUIDMap := readUIDMap(t, webPID)
	cacheUIDMap := readUIDMap(t, cachePID)

	// All should have the same UID mapping (same namespace)
	if dbUIDMap != webUIDMap {
		t.Errorf("web container has different UID mapping than anchor (db):\ndb: %s\nweb: %s", dbUIDMap, webUIDMap)
	}
	if dbUIDMap != cacheUIDMap {
		t.Errorf("cache container has different UID mapping than anchor (db):\ndb: %s\ncache: %s", dbUIDMap, cacheUIDMap)
	}

	// Verify the quadlet files have correct directives
	dbContainer := readQuadletFile(t, fmt.Sprintf("/etc/containers/systemd/cq-%s-db.container", project))
	if !strings.Contains(dbContainer, "--userns=auto:uidmapping=0:") {
		t.Errorf("anchor container (db) should have --userns=auto:uidmapping=0:<base>:65535, got:\n%s", dbContainer)
	}

	webContainer := readQuadletFile(t, fmt.Sprintf("/etc/containers/systemd/cq-%s-web.container", project))
	if !strings.Contains(webContainer, "--userns=container:") {
		t.Errorf("non-anchor container (web) should have --userns=container:<anchor>, got:\n%s", webContainer)
	}

	// Verify volumes have idmap option
	if !strings.Contains(dbContainer, "idmap") {
		t.Errorf("anchor container volumes should have idmap option, got:\n%s", dbContainer)
	}

	// Verify state has autons_range
	state := helpers.AssertProjectRegistered(t, project)
	if state.AutoNSRange == nil {
		t.Fatal("expected state to have autons_range")
	}
	if state.AutoNSRange.Size != 65535 {
		t.Errorf("expected autons_range size 65535, got %d", state.AutoNSRange.Size)
	}
	if state.AutoNSRange.Anchor != "db" {
		t.Errorf("expected anchor to be 'db', got %q", state.AutoNSRange.Anchor)
	}
}

func TestAutoNS_RootlessError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("this test verifies rootless error, but running as root")
	}

	project := helpers.ProjectName(t)
	compose := `
services:
  web:
    image: docker.io/library/alpine:latest
    command: sleep infinity
`
	dir, _ := helpers.WriteCompose(t, compose)

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Should fail in rootless mode
	result := helpers.ComquadWithEnv(t, dir, map[string]string{"COMQUAD_AUTONS": "1"}, "up", "--name", project)
	if result.ExitCode == 0 {
		t.Fatal("expected AutoNS to fail in rootless mode")
	}
	if !strings.Contains(result.Stderr, "rootful") {
		t.Errorf("expected error about rootful mode, got: %s", result.Stderr)
	}
}

func TestAutoNS_CheckCommand(t *testing.T) {
	skipIfNotRootful(t)
	skipIfNoContainersSubuid(t)

	// comquad check with COMQUAD_AUTONS=1 should pass when /etc/subuid is configured
	result := helpers.ComquadWithEnv(t, "", map[string]string{"COMQUAD_AUTONS": "1"}, "check")
	if result.ExitCode != 0 {
		t.Fatalf("comquad check with AutoNS failed:\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "containers entry found") {
		t.Errorf("expected check output to mention containers entry, got:\n%s", result.Stdout)
	}
}

func getContainerPID(t *testing.T, containerName string) string {
	t.Helper()
	cmd := exec.Command("podman", "inspect", "-f", "{{.State.Pid}}", containerName)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get PID for container %s: %v", containerName, err)
	}
	return strings.TrimSpace(string(out))
}

func readUIDMap(t *testing.T, pid string) string {
	t.Helper()
	data, err := os.ReadFile(fmt.Sprintf("/proc/%s/uid_map", pid))
	if err != nil {
		t.Fatalf("failed to read uid_map for PID %s: %v", pid, err)
	}
	return strings.TrimSpace(string(data))
}

func readQuadletFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read quadlet file %s: %v", path, err)
	}
	return string(data)
}
