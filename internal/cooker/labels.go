package cooker

import (
	"fmt"
	"strings"

	"comquad/internal/logger"
)

// addSystemdOptimizations adds [Install] sections and AutoUpdate where appropriate.
func (c *Cooker) addSystemdOptimizations(content string) string {
	lines := strings.Split(content, "\n")

	lines = append(lines, "", "[Install]", "WantedBy=default.target")

	hasNoAutoUpdate := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Label=comquad-no-autoupdate") {
			hasNoAutoUpdate = true
			break
		}
	}

	if !hasNoAutoUpdate {
		for i, line := range lines {
			if strings.TrimSpace(line) == "[Container]" {
				lines = append(lines[:i+1], append([]string{"AutoUpdate=registry"}, lines[i+1:]...)...)
				break
			}
		}
	}

	return strings.Join(lines, "\n")
}

// addProjectLabels injects comquad identity labels into quadlet files.
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
	lines = c.splitCombinedLabels(lines)

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

	hasProjectLabel := false
	hasManagedLabel := false
	for i := sectionIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && i > sectionIdx {
			break
		}
		if strings.HasPrefix(trimmed, "Label=com.comquad.project=") {
			hasProjectLabel = true
		}
		if strings.HasPrefix(trimmed, "Label=com.comquad.managed=") {
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

	insertAt := lastLabelIdx
	if insertAt < 0 {
		insertAt = sectionIdx
	}

	logger.Info(fmt.Sprintf("Added labels: Label=com.comquad.project=%s, Label=com.comquad.managed=true", c.ProjectName))

	lines = append(lines[:insertAt+1], append(labels, lines[insertAt+1:]...)...)

	return strings.Join(lines, "\n")
}

// addSELinuxLabels appends the :z SELinux label to all Volume= directives.
func (c *Cooker) addSELinuxLabels(content string) string {
	if !c.SELinuxEnabled {
		return content
	}

	changed := false
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "Volume=") {
			continue
		}
		value := strings.TrimPrefix(trimmed, "Volume=")
		newValue := c.addSELinuxToVolume(value)
		if newValue != value {
			lines[i] = "Volume=" + newValue
			changed = true
		}
	}

	if changed {
		logger.Info("Added SELinux :z labels to Volume= directives")
	}

	return strings.Join(lines, "\n")
}

// addSELinuxToVolume appends the ,z SELinux mount option to a Volume= directive value.
func (c *Cooker) addSELinuxToVolume(value string) string {
	parts := strings.SplitN(value, ":", 3)

	if len(parts) == 1 {
		return value + ":z"
	}

	if len(parts) == 2 {
		return value + ":z"
	}

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

// injectNetworkAliases adds NetworkAlias= directives to a container file.
func (c *Cooker) injectNetworkAliases(content string, fileName string) string {
	serviceName := strings.TrimPrefix(fileName, "cq-"+c.ProjectName+"-")
	serviceName = strings.TrimSuffix(serviceName, ".container")

	lines := strings.Split(content, "\n")

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

	containerName := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "ContainerName=") {
			containerName = strings.TrimPrefix(trimmed, "ContainerName=")
			break
		}
	}

	var aliases []string
	aliases = append(aliases, "NetworkAlias="+serviceName)
	if containerName != "" {
		aliases = append(aliases, "NetworkAlias="+containerName)
	}

	if len(aliases) == 0 {
		return content
	}

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
