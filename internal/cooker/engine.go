package cooker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Cooker handles the post-processing of Quadlet files
type Cooker struct {
	TempDir     string
	TargetDir   string
	ProjectName string
	IsRootless  bool
	PortOffset  int
}

// NewCooker creates a new cooker instance
func NewCooker(tempDir, targetDir, projectName string, isRootless bool, portOffset int) *Cooker {
	return &Cooker{
		TempDir:     tempDir,
		TargetDir:   targetDir,
		ProjectName: projectName,
		IsRootless:  isRootless,
		PortOffset:  portOffset,
	}
}

// Cook processes all files in the temp directory, renames them, and moves them to target
func (c *Cooker) Cook() error {
	entries, err := os.ReadDir(c.TempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	if err := os.MkdirAll(c.TargetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// First pass: build rename map so we know all old->new names
	renameMap := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		oldName := entry.Name()
		newName := c.buildNewFileName(oldName)
		renameMap[oldName] = newName
	}

	// Second pass: copy files with new names and rewrite internal references
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		oldName := entry.Name()
		newName := renameMap[oldName]

		srcPath := filepath.Join(c.TempDir, oldName)
		dstPath := filepath.Join(c.TargetDir, newName)

		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", oldName, err)
		}

		// Rewrite all internal references using the rename map
		updatedContent := c.rewriteReferences(string(content), renameMap)

		// Add systemd optimizations for .container and .network files
		if strings.HasSuffix(newName, ".container") || strings.HasSuffix(newName, ".network") {
			updatedContent = c.addSystemdOptimizations(updatedContent)
		}

		if err := os.WriteFile(dstPath, []byte(updatedContent), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", newName, err)
		}
	}

	// Apply port offsetting for rootless mode
	if c.IsRootless && c.PortOffset > 0 {
		if err := c.offsetPorts(); err != nil {
			return fmt.Errorf("failed to offset ports: %w", err)
		}
	}

	return nil
}

// buildNewFileName determines the new name for a file after cooking.
func (c *Cooker) buildNewFileName(oldName string) string {
	prefix := fmt.Sprintf("cq-%s-", c.ProjectName)

	// Already has our full prefix — don't double prefix
	if strings.HasPrefix(oldName, prefix) {
		return oldName
	}

	// Podlet added a generic "cq-" prefix — replace with our full prefix
	if strings.HasPrefix(oldName, "cq-") {
		return prefix + strings.TrimPrefix(oldName, "cq-")
	}

	// No prefix — add ours
	return prefix + oldName
}

// rewriteReferences rewrites quadlet unit references inside a file's content.
// It only replaces references in known quadlet directives (Network=, Volume=, etc.)
// to avoid corrupting image names or other values that share substrings with unit names.
func (c *Cooker) rewriteReferences(content string, renameMap map[string]string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Only rewrite lines that are quadlet cross-unit reference directives
		if !isReferenceDirective(trimmed) {
			continue
		}

		for oldName, newName := range renameMap {
			oldRef := stripQuadletExtension(oldName)
			newRef := stripQuadletExtension(newName)
			if oldRef != newRef {
				lines[i] = strings.ReplaceAll(lines[i], oldRef, newRef)
			}
		}
	}
	return strings.Join(lines, "\n")
}

// isReferenceDirective returns true for quadlet directives that reference other units.
func isReferenceDirective(line string) bool {
	directives := []string{
		"Network=",
		"Volume=",
		"Pod=",
	}
	for _, d := range directives {
		if strings.HasPrefix(line, d) {
			return true
		}
	}
	return false
}

// stripQuadletExtension removes known quadlet extensions from a filename.
func stripQuadletExtension(name string) string {
	for _, ext := range []string{".container", ".network", ".volume", ".pod", ".kube", ".image", ".build"} {
		if strings.HasSuffix(name, ext) {
			return strings.TrimSuffix(name, ext)
		}
	}
	return name
}

// addSystemdOptimizations adds [Install] sections and AutoUpdate where appropriate.
func (c *Cooker) addSystemdOptimizations(content string) string {
	lines := strings.Split(content, "\n")

	// Add [Install] section for containers and networks
	lines = append(lines, "", "[Install]", "WantedBy=default.target")

	// Check if AutoUpdate should be skipped
	hasNoAutoUpdate := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Label=comquad-no-autoupdate") {
			hasNoAutoUpdate = true
			break
		}
	}

	if !hasNoAutoUpdate {
		// Insert AutoUpdate=registry directly after [Container] header
		for i, line := range lines {
			if strings.TrimSpace(line) == "[Container]" {
				lines = append(lines[:i+1], append([]string{"AutoUpdate=registry"}, lines[i+1:]...)...)
				break
			}
		}
	}

	return strings.Join(lines, "\n")
}

// offsetPorts applies port offsetting for rootless containers.
// It offsets privileged ports (< 1024) by the PortOffset value,
// resolves internal conflicts within the project, and checks for external conflicts.
func (c *Cooker) offsetPorts() error {
	entries, err := os.ReadDir(c.TargetDir)
	if err != nil {
		return fmt.Errorf("failed to read target directory: %w", err)
	}

	// Collect all host ports across all container files
	type portEntry struct {
		filename string
		lineNum  int
		hostPort int
	}

	var allPorts []portEntry
	// Tracks which host ports are claimed in this project (after offsetting)
	claimedPorts := make(map[int]string) // port -> filename claiming it

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

	// Process each port: offset privileged ones, resolve conflicts
	for _, p := range allPorts {
		finalPort := p.hostPort

		// Only offset privileged ports (< 1024)
		if p.hostPort < 1024 {
			finalPort = p.hostPort + c.PortOffset
		}

		// Check for conflicts (internal to this project)
		if _, ok := claimedPorts[finalPort]; ok {
			// Internal conflict — resolve by incrementing
			for {
				finalPort++
				if _, exists := claimedPorts[finalPort]; !exists {
					break
				}
			}
		}

		// Claim this port
		claimedPorts[finalPort] = p.filename

		// Update the PublishPort line if the port changed
		if finalPort != p.hostPort {
			content, err := os.ReadFile(filepath.Join(c.TargetDir, p.filename))
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", p.filename, err)
			}
			lines := strings.Split(string(content), "\n")
			portStr := strings.TrimPrefix(strings.TrimSpace(lines[p.lineNum]), "PublishPort=")
			// Rebuild the port string with the new host port
			newPortStr := c.rebuildPublishPort(portStr, finalPort)
			lines[p.lineNum] = "PublishPort=" + newPortStr
			dstPath := filepath.Join(c.TargetDir, p.filename)
			if err := os.WriteFile(dstPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
				return fmt.Errorf("failed to update port in %s: %w", p.filename, err)
			}
		}
	}

	return nil
}

// rebuildPublishPort reconstructs a PublishPort value with a new host port.
// Preserves IP prefix and protocol suffix.
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
		// host:container
		return fmt.Sprintf("%d:%s", newHostPort, parts[1]) + protocol
	case 3:
		// ip:host:container
		return fmt.Sprintf("%s:%d:%s", parts[0], newHostPort, parts[2]) + protocol
	default:
		return fmt.Sprintf("%d", newHostPort) + protocol
	}
}

// parseHostPort extracts the host port from a PublishPort value.
// Handles formats: "host:container", "ip:host:container", "host:container/protocol"
func parseHostPort(portStr string) (int, error) {
	// Strip protocol suffix
	if strings.HasSuffix(portStr, "/tcp") {
		portStr = strings.TrimSuffix(portStr, "/tcp")
	} else if strings.HasSuffix(portStr, "/udp") {
		portStr = strings.TrimSuffix(portStr, "/udp")
	}

	parts := strings.Split(portStr, ":")
	var hostPortStr string

	switch len(parts) {
	case 1:
		// Just a port number
		hostPortStr = parts[0]
	case 2:
		// host:container
		hostPortStr = parts[0]
	case 3:
		// ip:host:container
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


