package cooker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"comquad/internal/logger"
)

// Cooker handles the post-processing of Quadlet files.
type Cooker struct {
	TempDir        string
	TargetDir      string
	ProjectName    string
	IsRootless     bool
	PortOffset     int
	SELinuxEnabled bool
}

// NewCooker creates a new cooker instance.
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

// Cook processes all files in the temp directory, renames them, and moves them to target.
func (c *Cooker) Cook() error {
	entries, err := os.ReadDir(c.TempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	if err := os.MkdirAll(c.TargetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

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

		original := string(content)
		updatedContent := c.rewriteReferences(original, renameMap)
		if updatedContent != original {
			logger.Info(fmt.Sprintf("Rewrote cross-unit references in %s", newName))
		}

		updatedContent = c.addSELinuxLabels(updatedContent)

		if strings.HasSuffix(newName, ".container") || strings.HasSuffix(newName, ".network") {
			updatedContent = c.addSystemdOptimizations(updatedContent)
			logger.Info(fmt.Sprintf("Added [Install] section to %s", newName))
			if strings.Contains(updatedContent, "AutoUpdate=registry") {
				logger.Info(fmt.Sprintf("Added AutoUpdate=registry to %s", newName))
			}
		}

		if strings.HasSuffix(newName, ".container") {
			updatedContent = c.injectNetworkAliases(updatedContent, newName)
		}

		updatedContent = c.addProjectLabels(updatedContent, newName)
		logger.Info(fmt.Sprintf("Added labels to %s", newName))

		if err := os.WriteFile(dstPath, []byte(updatedContent), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", newName, err)
		}
	}

	if c.IsRootless && c.PortOffset > 0 {
		if err := c.offsetPorts(); err != nil {
			return fmt.Errorf("failed to offset ports: %w", err)
		}
		logger.Action(fmt.Sprintf("Applied port offset %d for rootless mode", c.PortOffset))
	}

	return nil
}

// buildNewFileName determines the new name for a file after cooking.
func (c *Cooker) buildNewFileName(oldName string) string {
	prefix := fmt.Sprintf("cq-%s-", c.ProjectName)

	if strings.HasPrefix(oldName, prefix) {
		return oldName
	}

	if strings.HasPrefix(oldName, "cq-") {
		return prefix + strings.TrimPrefix(oldName, "cq-")
	}

	return prefix + oldName
}
