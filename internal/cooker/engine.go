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
}

// NewCooker creates a new cooker instance
func NewCooker(tempDir, targetDir, projectName string, isRootless bool) *Cooker {
	return &Cooker{
		TempDir:     tempDir,
		TargetDir:   targetDir,
		ProjectName: projectName,
		IsRootless:  isRootless,
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
