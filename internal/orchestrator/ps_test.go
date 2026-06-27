package orchestrator

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"comquad/internal/deploy"
)

func TestPs_HappyPathWithRunningContainers(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "active", subState: "running"},
		{name: "cq-myapp-db.service", activeState: "active", subState: "running"},
	}
	o := newTestOrchestrator("myapp", dir, state, sys)
	o.listContainers = func(projectName string, all bool) ([]ContainerInfo, error) {
		return []ContainerInfo{
			{
				Name:    "myapp-web",
				Image:   "docker.io/library/nginx:alpine",
				Command: "nginx -g daemon off;",
				Service: "web",
				State:   "running",
				Status:  "Up 2 minutes",
				DBusActive: "active",
				DBusSub:    "running",
			},
			{
				Name:    "myapp-db",
				Image:   "docker.io/library/postgres:15",
				Command: "postgres",
				Service: "db",
				State:   "running",
				Status:  "Up 2 minutes",
				DBusActive: "active",
				DBusSub:    "running",
			},
		}, nil
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := o.Ps(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("NAME")) {
		t.Errorf("expected output to contain 'NAME' header, got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("IMAGE")) {
		t.Errorf("expected output to contain 'IMAGE' header, got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("myapp-web")) {
		t.Errorf("expected output to contain 'myapp-web', got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("docker.io/library/nginx:alpine")) {
		t.Errorf("expected output to contain image, got:\n%s", output)
	}
}

func TestPs_EmptyProject(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	sys := newMockSystemdClient()
	o := newTestOrchestrator("myapp", dir, state, sys)
	o.listContainers = func(projectName string, all bool) ([]ContainerInfo, error) {
		return []ContainerInfo{}, nil
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := o.Ps(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "No containers found") {
		t.Errorf("expected 'No containers found' message, got:\n%s", output)
	}
}

func TestPs_ExitedContainersShowExitCode(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-worker.service", activeState: "inactive", subState: "dead"},
	}
	o := newTestOrchestrator("myapp", dir, state, sys)
	o.listContainers = func(projectName string, all bool) ([]ContainerInfo, error) {
		return []ContainerInfo{
			{
				Name:     "myapp-worker",
				Image:    "myapp-worker:latest",
				Command:  "python worker.py",
				Service:  "worker",
				State:    "exited",
				Status:   "Exited",
				ExitCode: 1,
			},
		}, nil
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := o.Ps(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Exited (1)") {
		t.Errorf("expected 'Exited (1)' in output, got:\n%s", output)
	}
}

func TestPs_ContainersWithPortsNetworksMounts(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "active", subState: "running"},
	}
	o := newTestOrchestrator("myapp", dir, state, sys)
	o.listContainers = func(projectName string, all bool) ([]ContainerInfo, error) {
		return []ContainerInfo{
			{
				Name:    "myapp-web",
				Image:   "docker.io/library/nginx:alpine",
				Command: "nginx -g daemon off;",
				Service: "web",
				State:   "running",
				Status:  "Up 2 minutes",
				Ports: []PortInfo{
					{Protocol: "tcp", ContainerPort: 80, HostIP: "", HostPort: 2080},
					{Protocol: "tcp", ContainerPort: 443, HostIP: "", HostPort: 2443},
				},
				Networks: []string{"systemd-cq-myapp-default"},
				Mounts:   []string{"/etc/nginx/nginx.conf", "/usr/share/nginx/html"},
				DBusActive: "active",
				DBusSub:    "running",
			},
		}, nil
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := o.Ps(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "0.0.0.0:2080->80/tcp") {
		t.Errorf("expected port mapping 2080->80, got:\n%s", output)
	}
	if !strings.Contains(output, "0.0.0.0:2443->443/tcp") {
		t.Errorf("expected port mapping 2443->443, got:\n%s", output)
	}
}
