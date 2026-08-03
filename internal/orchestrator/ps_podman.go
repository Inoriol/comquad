package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

func listContainersFromPodman(projectName string, all bool) ([]ContainerInfo, error) {
	args := []string{"ps", "--filter", "label=com.comquad.managed=true", "--filter", "label=com.comquad.project=" + projectName, "--format", "json"}
	if all {
		args = append(args, "-a")
	}
	cmd := exec.Command("podman", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run podman ps: %w", err)
	}

	var rawContainers []map[string]interface{}
	if err := json.Unmarshal(output, &rawContainers); err != nil {
		return nil, fmt.Errorf("failed to parse podman ps output: %w", err)
	}

	var containers []ContainerInfo
	for _, raw := range rawContainers {
		c := parseContainer(raw)
		if c == nil {
			continue
		}
		containers = append(containers, *c)
	}

	exposedPortsMap := batchGetExposedPorts(containers)

	for i := range containers {
		containers[i].ExposedPorts = exposedPortsMap[containers[i].Name]
	}
	return containers, nil
}

func parseContainer(raw map[string]interface{}) *ContainerInfo {
	if raw == nil {
		return nil
	}

	var name string
	if namesRaw, ok := raw["Names"].([]interface{}); ok && len(namesRaw) > 0 {
		if n, ok := namesRaw[0].(string); ok {
			name = n
		}
	}
	if name == "" {
		return nil
	}

	image, _ := raw["Image"].(string)
	commandRaw, _ := raw["Command"].([]interface{})
	var cmdParts []string
	for _, c := range commandRaw {
		if s, ok := c.(string); ok {
			cmdParts = append(cmdParts, s)
		}
	}
	command := strings.Join(cmdParts, " ")

	state, _ := raw["State"].(string)
	status, _ := raw["Status"].(string)

	exitCode := 0
	if ec, ok := raw["ExitCode"].(float64); ok {
		exitCode = int(ec)
	}

	created := time.Time{}
	if ca, ok := raw["Created"].(float64); ok {
		created = time.Unix(int64(ca), 0)
	}

	exitedAt := time.Time{}
	if ea, ok := raw["ExitedAt"].(float64); ok && ea > 0 {
		exitedAt = time.Unix(int64(ea), 0)
	}

	ports := parsePorts(raw["Ports"])
	networks := parseStringSlice(raw["Networks"])
	mounts := parseStringSlice(raw["Mounts"])

	// Derive service name from container name by stripping cq-<project>- prefix
	service := name
	if idx := strings.Index(name, "-"); idx > 0 {
		service = name[idx+1:]
	}

	return &ContainerInfo{
		Name:      name,
		Image:     image,
		Command:   command,
		Service:   service,
		State:     state,
		Status:    status,
		ExitCode:  exitCode,
		CreatedAt: created,
		ExitedAt:  exitedAt,
		Ports:     ports,
		Networks:  networks,
		Mounts:    mounts,
	}
}

func parsePorts(raw interface{}) []PortInfo {
	if raw == nil {
		return nil
	}
	rawPorts, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var ports []PortInfo
	for _, rp := range rawPorts {
		m, ok := rp.(map[string]interface{})
		if !ok {
			continue
		}
		p := PortInfo{}
		if proto, ok := m["protocol"].(string); ok {
			p.Protocol = proto
		}
		if cp, ok := m["container_port"].(float64); ok {
			p.ContainerPort = int(cp)
		}
		if ip, ok := m["host_ip"].(string); ok {
			p.HostIP = ip
		}
		if hp, ok := m["host_port"].(float64); ok {
			p.HostPort = int(hp)
		}
		ports = append(ports, p)
	}
	return ports
}

func parseStringSlice(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	rawSlice, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var result []string
	for _, item := range rawSlice {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func batchGetExposedPorts(containers []ContainerInfo) map[string][]string {
	if len(containers) == 0 {
		return nil
	}

	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = c.Name
	}

	args := append([]string{"inspect"}, names...)
	cmd := exec.Command("podman", args...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "comquad: warning: failed to inspect containers: %v\n", err)
		return nil
	}

	var inspectResults []struct {
		Config struct {
			ExposedPorts map[string]interface{} `json:"ExposedPorts"`
		} `json:"Config"`
		Name string `json:"Name"`
	}
	if err := json.Unmarshal(output, &inspectResults); err != nil {
		fmt.Fprintf(os.Stderr, "comquad: warning: failed to parse inspect output: %v\n", err)
		return nil
	}

	result := make(map[string][]string, len(inspectResults))
	for _, r := range inspectResults {
		if r.Name == "" || r.Config.ExposedPorts == nil {
			continue
		}
		var ports []string
		for portKey := range r.Config.ExposedPorts {
			ports = append(ports, portKey)
		}
		sort.Strings(ports)
		result[r.Name] = ports
	}
	return result
}
