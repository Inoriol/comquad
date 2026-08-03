package cooker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"comquad/internal/logger"
)

// offsetPorts applies port offsetting for rootless containers.
func (c *Cooker) offsetPorts() error {
	entries, err := os.ReadDir(c.TargetDir)
	if err != nil {
		return fmt.Errorf("failed to read target directory: %w", err)
	}

	type portEntry struct {
		filename string
		lineNum  int
		hostPort int
	}

	var allPorts []portEntry
	fileLines := make(map[string][]string)
	claimedPorts := make(map[int]string)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".container") {
			continue
		}

		filename := entry.Name()
		content, err := os.ReadFile(filepath.Join(c.TargetDir, filename))
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filename, err)
		}

		lines := strings.Split(string(content), "\n")
		fileLines[filename] = lines
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "PublishPort=") {
				continue
			}

			portStr := strings.TrimPrefix(trimmed, "PublishPort=")
			hostPort, err := parseHostPort(portStr)
			if err != nil {
				return fmt.Errorf("failed to parse port %q in %s line %d: %w", portStr, filename, i+1, err)
			}

			allPorts = append(allPorts, portEntry{
				filename: filename,
				lineNum:  i,
				hostPort: hostPort,
			})
		}
	}

	for _, p := range allPorts {
		finalPort := p.hostPort

		if p.hostPort < 1024 {
			finalPort = p.hostPort + c.PortOffset
		}

		if _, ok := claimedPorts[finalPort]; ok {
			for {
				finalPort++
				if finalPort > 65535 {
					return fmt.Errorf("no available port above %d (all ports in range are claimed)", p.hostPort)
				}
				if _, exists := claimedPorts[finalPort]; !exists {
					break
				}
			}
		}

		claimedPorts[finalPort] = p.filename

		if finalPort != p.hostPort {
			lines := fileLines[p.filename]
			portStr := strings.TrimPrefix(strings.TrimSpace(lines[p.lineNum]), "PublishPort=")
			newPortStr := c.rebuildPublishPort(portStr, finalPort)
			lines[p.lineNum] = "PublishPort=" + newPortStr
			dstPath := filepath.Join(c.TargetDir, p.filename)
			if err := os.WriteFile(dstPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
				return fmt.Errorf("failed to update port in %s: %w", p.filename, err)
			}
			logger.Action(fmt.Sprintf("Offset port in %s: %s → %s", p.filename, portStr, newPortStr))
		}
	}

	return nil
}

// rebuildPublishPort reconstructs a PublishPort value with a new host port.
func (c *Cooker) rebuildPublishPort(portStr string, newHostPort int) string {
	protocol := ""
	if strings.HasSuffix(portStr, "/tcp") {
		protocol = "/tcp"
	} else if strings.HasSuffix(portStr, "/udp") {
		protocol = "/udp"
	}
	portStr = strings.TrimSuffix(portStr, "/tcp")
	portStr = strings.TrimSuffix(portStr, "/udp")

	parts := strings.Split(portStr, ":")
	switch len(parts) {
	case 2:
		return fmt.Sprintf("%d:%s", newHostPort, parts[1]) + protocol
	case 3:
		return fmt.Sprintf("%s:%d:%s", parts[0], newHostPort, parts[2]) + protocol
	default:
		return fmt.Sprintf("%d", newHostPort) + protocol
	}
}

// parseHostPort extracts the host port from a PublishPort value.
func parseHostPort(portStr string) (int, error) {
	if strings.HasSuffix(portStr, "/tcp") {
		portStr = strings.TrimSuffix(portStr, "/tcp")
	} else if strings.HasSuffix(portStr, "/udp") {
		portStr = strings.TrimSuffix(portStr, "/udp")
	}

	parts := strings.Split(portStr, ":")
	var hostPortStr string

	switch len(parts) {
	case 1:
		hostPortStr = parts[0]
	case 2:
		hostPortStr = parts[0]
	case 3:
		hostPortStr = parts[1]
	default:
		return 0, fmt.Errorf("invalid port format: %s", portStr)
	}

	var hostPort int
	_, err := fmt.Sscanf(hostPortStr, "%d", &hostPort)
	if err != nil {
		return 0, fmt.Errorf("invalid host port %q: %w", hostPortStr, err)
	}

	return hostPort, nil
}
