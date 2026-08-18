package orchestrator

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Inoriol/comquad/internal/deploy"
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
				Name:       "myapp-web",
				Image:      "docker.io/library/nginx:alpine",
				Command:    "nginx -g daemon off;",
				Service:    "web",
				State:      "running",
				Status:     "Up 2 minutes",
				DBusActive: "active",
				DBusSub:    "running",
			},
			{
				Name:       "myapp-db",
				Image:      "docker.io/library/postgres:15",
				Command:    "postgres",
				Service:    "db",
				State:      "running",
				Status:     "Up 2 minutes",
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

func TestParseContainer_UsesProjectPrefixForServiceName(t *testing.T) {
	raw := map[string]interface{}{
		"Names":  []interface{}{"my-app-web"},
		"State":  "running",
		"Status": "Up 1 minute",
	}

	container := parseContainer(raw, "my-app")
	if container == nil {
		t.Fatal("expected container")
	}
	if container.Service != "web" {
		t.Fatalf("expected service web, got %q", container.Service)
	}
}

func TestPs_MergesSystemdState(t *testing.T) {
	dir := t.TempDir()
	state := newMockStateStore(map[string]deploy.ProjectState{
		"myapp": makeProjectState("myapp", dir, nil),
	})
	sys := newMockSystemdClient()
	sys.units = []unitRecord{
		{name: "cq-myapp-web.service", activeState: "active", subState: "running"},
	}
	o := newTestOrchestrator("myapp", dir, state, sys)

	var returned []ContainerInfo
	o.listContainers = func(projectName string, all bool) ([]ContainerInfo, error) {
		returned = []ContainerInfo{{Name: "myapp-web", Image: "nginx:latest", State: "running", Status: "Up 1m"}}
		return returned, nil
	}

	if err := o.Ps(false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(returned) != 1 {
		t.Fatalf("expected 1 container, got %d", len(returned))
	}
	if returned[0].DBusActive != "active" || returned[0].DBusSub != "running" {
		t.Errorf("expected systemd state merged, got active=%q sub=%q", returned[0].DBusActive, returned[0].DBusSub)
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
				Networks:   []string{"systemd-cq-myapp-default"},
				Mounts:     []string{"/etc/nginx/nginx.conf", "/usr/share/nginx/html"},
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

func TestPs_WithExposedPorts(t *testing.T) {
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
				Name:         "myapp-web",
				Image:        "docker.io/library/nginx:alpine",
				Command:      "nginx -g daemon off;",
				Service:      "web",
				State:        "running",
				Status:       "Up 2 minutes",
				ExposedPorts: []string{"8080/tcp", "8443/tcp"},
				DBusActive:   "active",
				DBusSub:      "running",
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

	if !strings.Contains(output, "8080/tcp") {
		t.Errorf("expected exposed port 8080/tcp in output, got:\n%s", output)
	}
	if !strings.Contains(output, "8443/tcp") {
		t.Errorf("expected exposed port 8443/tcp in output, got:\n%s", output)
	}
}

func TestPs_WithExposedAndPublishedPorts(t *testing.T) {
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
				Name:         "myapp-web",
				Image:        "docker.io/library/nginx:alpine",
				Command:      "nginx -g daemon off;",
				Service:      "web",
				State:        "running",
				Status:       "Up 2 minutes",
				ExposedPorts: []string{"8080/tcp"},
				Ports: []PortInfo{
					{Protocol: "tcp", ContainerPort: 80, HostIP: "", HostPort: 2080},
				},
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

	if !strings.Contains(output, "8080/tcp") {
		t.Errorf("expected exposed port 8080/tcp in output, got:\n%s", output)
	}
	if !strings.Contains(output, "0.0.0.0:2080->80/tcp") {
		t.Errorf("expected published port 0.0.0.0:2080->80/tcp in output, got:\n%s", output)
	}
}

func TestFormatTimeAgo_Now(t *testing.T) {
	result := formatTimeAgo(time.Now())
	if result != "just now" {
		t.Errorf("expected 'just now', got %q", result)
	}
}

func TestFormatTimeAgo_Seconds(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-30 * time.Second))
	if result != "30s ago" {
		t.Errorf("expected '30s ago', got %q", result)
	}
}

func TestFormatTimeAgo_OneMinute(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-1 * time.Minute))
	if result != "1m ago" {
		t.Errorf("expected '1m ago', got %q", result)
	}
}

func TestFormatTimeAgo_Minutes(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-5 * time.Minute))
	if result != "5m ago" {
		t.Errorf("expected '5m ago', got %q", result)
	}
}

func TestFormatTimeAgo_OneHour(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-1 * time.Hour))
	if result != "1h ago" {
		t.Errorf("expected '1h ago', got %q", result)
	}
}

func TestFormatTimeAgo_Hours(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-3 * time.Hour))
	if result != "3h ago" {
		t.Errorf("expected '3h ago', got %q", result)
	}
}

func TestFormatTimeAgo_OneDay(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-24 * time.Hour))
	if result != "1d ago" {
		t.Errorf("expected '1d ago', got %q", result)
	}
}

func TestFormatTimeAgo_Days(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-5 * 24 * time.Hour))
	if result != "5d ago" {
		t.Errorf("expected '5d ago', got %q", result)
	}
}

func TestFormatTimeAgo_OneWeek(t *testing.T) {
	tm := time.Now().Add(-8 * 24 * time.Hour)
	result := formatTimeAgo(tm)
	expected := tm.Format("Jan 02 2006")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatTimeAgo_Future(t *testing.T) {
	tm := time.Now().Add(1 * time.Hour)
	result := formatTimeAgo(tm)
	expected := tm.Format("Jan 02 2006")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatPorts_ExposedOnly(t *testing.T) {
	result := formatPorts(nil, []string{"80/tcp", "443/tcp"})
	if result != "80/tcp, 443/tcp" {
		t.Errorf("expected '80/tcp, 443/tcp', got %q", result)
	}
}

func TestFormatPorts_PublishedOnly(t *testing.T) {
	ports := []PortInfo{
		{HostIP: "0.0.0.0", HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
	}
	result := formatPorts(ports, nil)
	if result != "0.0.0.0:8080->80/tcp" {
		t.Errorf("expected '0.0.0.0:8080->80/tcp', got %q", result)
	}
}

func TestFormatPorts_DefaultHostIP(t *testing.T) {
	ports := []PortInfo{
		{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
	}
	result := formatPorts(ports, nil)
	if result != "0.0.0.0:8080->80/tcp" {
		t.Errorf("expected '0.0.0.0:8080->80/tcp', got %q", result)
	}
}

func TestFormatPorts_MixedExposedAndPublished(t *testing.T) {
	ports := []PortInfo{
		{HostIP: "0.0.0.0", HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
	}
	result := formatPorts(ports, []string{"443/tcp"})
	if result != "443/tcp, 0.0.0.0:8080->80/tcp" {
		t.Errorf("expected '443/tcp, 0.0.0.0:8080->80/tcp', got %q", result)
	}
}

func TestFormatCreated_Running(t *testing.T) {
	now := time.Now()
	result := formatCreated(now, time.Time{}, "running")
	if result != formatTimeAgo(now) {
		t.Errorf("expected %q, got %q", formatTimeAgo(now), result)
	}
}

func TestFormatCreated_Exited(t *testing.T) {
	created := time.Now().Add(-1 * time.Hour)
	exited := time.Now().Add(-30 * time.Minute)
	result := formatCreated(created, exited, "exited")
	expected := formatTimeAgo(exited)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatCreated_ExitedNoExitTime(t *testing.T) {
	created := time.Now().Add(-1 * time.Hour)
	result := formatCreated(created, time.Time{}, "exited")
	expected := formatTimeAgo(created)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestParseContainer_BasicFields(t *testing.T) {
	raw := map[string]interface{}{
		"Names":    []interface{}{"myapp-web"},
		"Image":    "docker.io/library/nginx:latest",
		"Command":  []interface{}{"nginx", "-g", "daemon off;"},
		"State":    "running",
		"Status":   "Up 5 minutes",
		"ExitCode": float64(0),
		"Created":  float64(1700000000),
		"ExitedAt": float64(0),
		"Ports":    nil,
		"Networks": nil,
		"Mounts":   nil,
	}

	c := parseContainer(raw)
	if c == nil {
		t.Fatal("expected non-nil container")
	}
	if c.Name != "myapp-web" {
		t.Errorf("expected name 'myapp-web', got %q", c.Name)
	}
	if c.Image != "docker.io/library/nginx:latest" {
		t.Errorf("expected image, got %q", c.Image)
	}
	if c.Command != "nginx -g daemon off;" {
		t.Errorf("expected command, got %q", c.Command)
	}
	if c.State != "running" {
		t.Errorf("expected state 'running', got %q", c.State)
	}
	if c.Status != "Up 5 minutes" {
		t.Errorf("expected status, got %q", c.Status)
	}
	if c.Service != "web" {
		t.Errorf("expected service 'web', got %q", c.Service)
	}
}

func TestParseContainer_ServiceNameDerivation(t *testing.T) {
	raw := map[string]interface{}{
		"Names":    []interface{}{"cq-myproject-web"},
		"Command":  []interface{}{},
		"Created":  float64(1700000000),
		"ExitedAt": float64(0),
	}

	c := parseContainer(raw)
	if c == nil {
		t.Fatal("expected non-nil container")
	}
	if c.Service != "myproject-web" {
		t.Errorf("expected service 'myproject-web', got %q", c.Service)
	}
}

func TestParseContainer_ExitedContainer(t *testing.T) {
	raw := map[string]interface{}{
		"Names":    []interface{}{"myapp-web"},
		"Command":  []interface{}{},
		"State":    "exited",
		"Status":   "Exited (0) 2 minutes ago",
		"ExitCode": float64(0),
		"Created":  float64(1700000000),
		"ExitedAt": float64(1700000100),
	}

	c := parseContainer(raw)
	if c == nil {
		t.Fatal("expected non-nil container")
	}
	if c.State != "exited" {
		t.Errorf("expected state 'exited', got %q", c.State)
	}
	if c.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", c.ExitCode)
	}
	if c.ExitedAt.IsZero() {
		t.Error("expected non-zero exited at time")
	}
}

func TestParseContainer_NilInput(t *testing.T) {
	c := parseContainer(nil)
	if c != nil {
		t.Error("expected nil for nil input")
	}
}

func TestParseContainer_EmptyName(t *testing.T) {
	raw := map[string]interface{}{
		"Names": []interface{}{},
	}
	c := parseContainer(raw)
	if c != nil {
		t.Error("expected nil for empty name")
	}
}

func TestParseContainer_NoNamesField(t *testing.T) {
	raw := map[string]interface{}{
		"Image": "nginx",
	}
	c := parseContainer(raw)
	if c != nil {
		t.Error("expected nil when Names field missing")
	}
}

func TestParsePorts_ValidPorts(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"container_port": float64(80),
			"host_ip":        "0.0.0.0",
			"host_port":      float64(8080),
			"protocol":       "tcp",
		},
	}

	ports := parsePorts(raw)
	if len(ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(ports))
	}
	if ports[0].ContainerPort != 80 {
		t.Errorf("expected container port 80, got %d", ports[0].ContainerPort)
	}
	if ports[0].HostPort != 8080 {
		t.Errorf("expected host port 8080, got %d", ports[0].HostPort)
	}
	if ports[0].HostIP != "0.0.0.0" {
		t.Errorf("expected host IP '0.0.0.0', got %q", ports[0].HostIP)
	}
	if ports[0].Protocol != "tcp" {
		t.Errorf("expected protocol 'tcp', got %q", ports[0].Protocol)
	}
}

func TestParsePorts_Nil(t *testing.T) {
	ports := parsePorts(nil)
	if ports != nil {
		t.Error("expected nil for nil input")
	}
}

func TestParsePorts_WrongType(t *testing.T) {
	ports := parsePorts("not a slice")
	if ports != nil {
		t.Error("expected nil for wrong type")
	}
}

func TestParseStringSlice_Valid(t *testing.T) {
	raw := []interface{}{"eth0", "eth1"}
	result := parseStringSlice(raw)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0] != "eth0" || result[1] != "eth1" {
		t.Errorf("expected ['eth0', 'eth1'], got %v", result)
	}
}

func TestParseStringSlice_Nil(t *testing.T) {
	result := parseStringSlice(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestParseStringSlice_WrongType(t *testing.T) {
	result := parseStringSlice("not a slice")
	if result != nil {
		t.Error("expected nil for wrong type")
	}
}

func TestParseStringSlice_MixedTypes(t *testing.T) {
	raw := []interface{}{"hello", 42}
	result := parseStringSlice(raw)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0] != "hello" {
		t.Errorf("expected 'hello', got %q", result[0])
	}
}
