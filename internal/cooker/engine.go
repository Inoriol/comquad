package cooker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"comquad/internal/logger"
)

// Cooker handles the post-processing of Quadlet files
type Cooker struct {
	TempDir         string
	TargetDir       string
	ProjectName     string
	IsRootless      bool
	PortOffset      int
	SELinuxEnabled  bool
}

// NewCooker creates a new cooker instance
func NewCooker(tempDir, targetDir, projectName string, isRootless bool, portOffset int, selinuxEnabled bool) *Cooker {
	return &Cooker{
		TempDir:        tempDir,
		TargetDir:      targetDir,
		ProjectName:    projectName,
		IsRootless:     isRootless,
		PortOffset:     portOffset,
		SELinuxEnabled: selinuxEnabled,
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
		logger.Info(fmt.Sprintf("Renamed %s → %s", oldName, newName))
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
		original := string(content)
		updatedContent := c.rewriteReferences(original, renameMap)
		if updatedContent != original {
			logger.Info(fmt.Sprintf("Rewrote cross-unit references in %s", newName))
		}

		// Add SELinux :z label to Volume= directives
		updatedContent = c.addSELinuxLabels(updatedContent)

		// Add systemd optimizations for .container and .network files
		if strings.HasSuffix(newName, ".container") || strings.HasSuffix(newName, ".network") {
			updatedContent = c.addSystemdOptimizations(updatedContent)
			logger.Info(fmt.Sprintf("Added [Install] section to %s", newName))
			if strings.Contains(updatedContent, "AutoUpdate=registry") {
				logger.Info(fmt.Sprintf("Added AutoUpdate=registry to %s", newName))
			}
		}

		// Add NetworkAlias= for DNS resolution within compose networks
		if strings.HasSuffix(newName, ".container") {
			updatedContent = c.injectNetworkAliases(updatedContent, newName)
		}

		// Add labels to all file types
		updatedContent = c.addProjectLabels(updatedContent, newName)
		logger.Info(fmt.Sprintf("Added labels to %s", newName))

		if err := os.WriteFile(dstPath, []byte(updatedContent), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", newName, err)
		}
	}

	// Apply port offsetting for rootless mode
	if c.IsRootless && c.PortOffset > 0 {
		if err := c.offsetPorts(); err != nil {
			return fmt.Errorf("failed to offset ports: %w", err)
		}
		logger.Info(fmt.Sprintf("Applied port offset %d for rootless mode", c.PortOffset))
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

		// Sort keys longest-first to avoid partial prefix matches
		sortedKeys := make([]string, 0, len(renameMap))
		for k := range renameMap {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Slice(sortedKeys, func(i, j int) bool {
			return len(sortedKeys[i]) > len(sortedKeys[j])
		})
		for _, oldName := range sortedKeys {
			newName := renameMap[oldName]
			oldRef := stripQuadletExtension(oldName)
			newRef := stripQuadletExtension(newName)
			if oldRef != newRef {
				lines[i] = c.replaceDirectiveValue(lines[i], oldRef, newRef)
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
		"Wants=",
		"Requires=",
		"Requisite=",
		"BindsTo=",
		"PartOf=",
		"Upholds=",
		"Conflicts=",
		"Before=",
		"After=",
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
	for _, ext := range []string{".container", ".service", ".network", ".volume", ".pod", ".kube", ".image", ".build"} {
		if strings.HasSuffix(name, ext) {
			return strings.TrimSuffix(name, ext)
		}
	}
	return name
}

// replaceDirectiveValue replaces oldRef with newRef in a quadlet directive value,
// respecting the semantics of each directive type. For Volume= only the volume
// name (first colon-delimited component) is replaced; container paths are left
// untouched. For Network= and Pod= the entire value is replaced since they hold
// only a unit name. For [Unit] dependency directives (After=, Requires=, etc.)
// the value is split by spaces and each token is replaced individually, preserving
// its suffix (.container, .service, etc.).
func (c *Cooker) replaceDirectiveValue(line, oldRef, newRef string) string {
	colonIdx := strings.Index(line, "=")
	if colonIdx < 0 {
		return line
	}
	directive := line[:colonIdx]
	value := line[colonIdx+1:]

	// [Unit] dependency directives can have multiple space-separated unit references.
	unitDirectives := map[string]bool{
		"Wants":     true,
		"Requires":  true,
		"Requisite": true,
		"BindsTo":   true,
		"PartOf":    true,
		"Upholds":   true,
		"Conflicts": true,
		"Before":    true,
		"After":     true,
	}

	if unitDirectives[directive] {
		return c.replaceUnitDirectives(line, directive, value, oldRef, newRef)
	}

	switch directive {
	case "Volume":
		parts := strings.SplitN(value, ":", 2)
		// If the first part contains '/', it's a host path (bind mount), not a volume name.
		// Never touch host paths — only replace named volume references.
		if len(parts) >= 2 && strings.Contains(parts[0], "/") {
			return line
		}
		if strings.Contains(parts[0], oldRef) {
			// Skip if the new reference is already present (prevents double-prefixing)
			if strings.Contains(parts[0], newRef) {
				return line
			}
			replaced := strings.Replace(parts[0], oldRef, newRef, 1)
			if len(parts) == 2 {
				return directive + "=" + replaced + ":" + parts[1]
			}
			return directive + "=" + replaced
		}
		return line
	default:
		// Skip if the new reference is already present (prevents double-prefixing)
		if strings.Contains(value, newRef) {
			return line
		}
		return directive + "=" + strings.Replace(value, oldRef, newRef, 1)
	}
}

// replaceUnitDirectives handles [Unit] section directives with multiple space-separated unit references.
// It replaces oldRef with newRef in each token while preserving the token's suffix (.container, .service, etc.).
func (c *Cooker) replaceUnitDirectives(line, directive, value, oldRef, newRef string) string {
	if oldRef == newRef {
		return line
	}

	tokens := strings.Fields(value)
	if len(tokens) == 0 {
		return line
	}

	changed := false
	for i, token := range tokens {
		// Strip extension from the token for exact matching
		tokenName := stripQuadletExtension(token)
		if tokenName == oldRef {
			// Skip if the new reference is already present in this token
			if strings.Contains(token, newRef) {
				continue
			}
			// Preserve the original suffix
			ext := ""
			if strings.HasSuffix(token, ".service") {
				ext = ".service"
			} else if strings.HasSuffix(token, ".container") {
				ext = ".container"
			}
			tokens[i] = newRef + ext
			changed = true
		}
	}

	if !changed {
		return line
	}

	return directive + "=" + strings.Join(tokens, " ")
}

// splitCombinedLabels splits combined Label= lines into separate Label= lines.
// e.g. "Label=a=b c=d" → "Label=a=b\nLabel=c=d"
func (c *Cooker) splitCombinedLabels(lines []string) []string {
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Label=") {
			value := strings.TrimPrefix(trimmed, "Label=")
			pairs := strings.Fields(value)
			for _, pair := range pairs {
				result = append(result, "Label="+pair)
			}
		} else {
			result = append(result, line)
		}
	}
	return result
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
				if finalPort > 65535 {
					return fmt.Errorf("no available port above %d (all ports in range are claimed)", p.hostPort)
				}
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
			logger.Info(fmt.Sprintf("Offset port in %s: %s → %s", p.filename, portStr, newPortStr))
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

// addProjectLabels injects comquad identity labels into quadlet files.
// All files get Label=com.comquad.managed=true.
// .network and .volume files also get Label=com.comquad.project=<projectName>.
func (c *Cooker) addProjectLabels(content string, fileName string) string {
	var sectionHeader string

	switch {
	case strings.HasSuffix(fileName, ".container"):
		sectionHeader = "[Container]"
	case strings.HasSuffix(fileName, ".network"):
		sectionHeader = "[Network]"
	case strings.HasSuffix(fileName, ".volume"):
		sectionHeader = "[Volume]"
	default:
		return content
	}

	lines := strings.Split(content, "\n")

	// Split combined Label= lines (e.g. "Label=a=b c=d") into separate lines.
	lines = c.splitCombinedLabels(lines)

	// Find the section header
	sectionIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == sectionHeader {
			sectionIdx = i
			break
		}
	}

	if sectionIdx < 0 {
		return content
	}

	// Find the last Label= line in this section
	lastLabelIdx := -1
	for i := sectionIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && i > sectionIdx {
			break
		}
		if strings.HasPrefix(trimmed, "Label=") {
			lastLabelIdx = i
		}
	}

	// Check which labels are already present to avoid duplicates on re-deploy.
	hasProjectLabel := false
	hasManagedLabel := false
	for i := sectionIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && i > sectionIdx {
			break
		}
		if trimmed == "Label=com.comquad.project="+c.ProjectName {
			hasProjectLabel = true
		}
		if trimmed == "Label=com.comquad.managed=true" {
			hasManagedLabel = true
		}
	}

	var labels []string
	if !hasProjectLabel {
		labels = append(labels, "Label=com.comquad.project="+c.ProjectName)
	}
	if !hasManagedLabel {
		labels = append(labels, "Label=com.comquad.managed=true")
	}

if len(labels) == 0 {
		return strings.Join(lines, "\n")
	}

	// Insert new labels after the last Label= line (or after section header)
	insertAt := lastLabelIdx
	if insertAt < 0 {
		insertAt = sectionIdx
	}

	logger.Info(fmt.Sprintf("Added labels: Label=com.comquad.project=%s, Label=com.comquad.managed=true", c.ProjectName))

	lines = append(lines[:insertAt+1], append(labels, lines[insertAt+1:]...)...)

	return strings.Join(lines, "\n")
}

// addSELinuxLabels appends the :z SELinux label to all Volume= directives
// in the content when SELinux is enabled on the host.
func (c *Cooker) addSELinuxLabels(content string) string {
	if !c.SELinuxEnabled {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "Volume=") {
			continue
		}
		value := strings.TrimPrefix(trimmed, "Volume=")
		lines[i] = "Volume=" + c.addSELinuxToVolume(value)
	}

	return strings.Join(lines, "\n")
}

// addSELinuxToVolume appends the ,z SELinux mount option to a Volume= directive value.
// It handles all cases: no options, :ro, :rw, already has :z/:Z, etc.
func (c *Cooker) addSELinuxToVolume(value string) string {
	parts := strings.SplitN(value, ":", 3)

	// No colon at all (e.g., "appvol") — add :z
	if len(parts) == 1 {
		return value + ":z"
	}

	// No options present — add :z
	if len(parts) == 2 {
		return value + ":z"
	}

	// Options present — check if already has z or Z as a comma-delimited token
	if len(parts) == 3 {
		options := strings.Split(parts[2], ",")
		for _, opt := range options {
			if opt == "z" || opt == "Z" {
				return value
			}
		}
		return parts[0] + ":" + parts[1] + ":" + parts[2] + ",z"
	}

	return value
}

// injectNetworkAliases adds NetworkAlias= directives to a container file
// so that the service can be resolved by name within compose networks.
// It always adds an alias for the service name, and a second one for
// ContainerName if present (matching docker compose DNS behavior).
func (c *Cooker) injectNetworkAliases(content string, fileName string) string {
	// Extract service name from filename: cq-<project>-<service>.container
	serviceName := strings.TrimPrefix(fileName, "cq-"+c.ProjectName+"-")
	serviceName = strings.TrimSuffix(serviceName, ".container")

	lines := strings.Split(content, "\n")

	// Find [Container] section
	sectionIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "[Container]" {
			sectionIdx = i
			break
		}
	}

	if sectionIdx < 0 {
		return content
	}

	// Check if NetworkAlias already exists (idempotent)
	hasNetworkAlias := false
	for i := sectionIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && i > sectionIdx {
			break
		}
		if strings.HasPrefix(trimmed, "NetworkAlias=") {
			hasNetworkAlias = true
			break
		}
	}

	if hasNetworkAlias {
		return content
	}

	// Find ContainerName= for second alias
	containerName := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "ContainerName=") {
			containerName = strings.TrimPrefix(trimmed, "ContainerName=")
			break
		}
	}

	// Collect aliases to inject
	var aliases []string
	aliases = append(aliases, "NetworkAlias="+serviceName)
	if containerName != "" {
		aliases = append(aliases, "NetworkAlias="+containerName)
	}

	if len(aliases) == 0 {
		return content
	}

	// Find last NetworkAlias= or last line of [Container] section
	insertAt := sectionIdx
	for i := sectionIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && i > sectionIdx {
			break
		}
		if strings.HasPrefix(trimmed, "NetworkAlias=") {
			insertAt = i
		}
	}

	logger.Info(fmt.Sprintf("Added NetworkAlias=%s to %s", strings.Join(aliases, ","), fileName))

	lines = append(lines[:insertAt+1], append(aliases, lines[insertAt+1:]...)...)

	return strings.Join(lines, "\n")
}


