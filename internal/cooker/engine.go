package cooker

import (
        "fmt"
        "os"
        "path/filepath"
        "strings"
)

const rootlessPortOffset = 8080

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

                if err := os.WriteFile(dstPath, []byte(updatedContent), 0644); err != nil {
                        return fmt.Errorf("failed to write file %s: %w", newName, err)
                }
        }

        return nil
}

// buildNewFileName determines the new name for a file after cooking.
func (c *Cooker) buildNewFileName(oldName string) string {
        prefix := fmt.Sprintf("comquad-%s-", c.ProjectName)

        // Already has our full prefix — don't double prefix
        if strings.HasPrefix(oldName, prefix) {
                return oldName
        }

        // Podlet added a generic "comquad-" prefix — replace with our full prefix
        if strings.HasPrefix(oldName, "comquad-") {
                return prefix + strings.TrimPrefix(oldName, "comquad-")
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
                // ❌ Remove ContainerName= — it's a name, not a cross-unit reference
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

func (c *Cooker) isSELinuxEnforcing() (bool, error) {
        data, err := os.ReadFile("/sys/fs/selinux/enforce")
        if err != nil {
                return false, nil
        }
        return strings.TrimSpace(string(data)) == "1", nil
}

func (c *Cooker) processAndMove(oldPath, newPath string, selinuxEnforcing bool) error {
        content, err := os.ReadFile(oldPath)
        if err != nil {
                return err
        }

        lines := strings.Split(string(content), "\n")
        var newLines []string

        // Track current section for context-aware processing
        currentSection := ""

        for _, line := range lines {
                trimmed := strings.TrimSpace(line)

                // Skip comments entirely — do not modify them
                if strings.HasPrefix(trimmed, "#") {
                        newLines = append(newLines, line)
                        continue
                }

                // Track which section we are in e.g. [Container], [Service], [Network]
                if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
                        currentSection = trimmed
                        newLines = append(newLines, line)
                        continue
                }

                newLine := line

                // 1. SELinux: Append relabel=private to Volume= lines only inside [Container]
                // Scoped to [Container] section to avoid mismatching [Service] or other sections
                if selinuxEnforcing &&
                        currentSection == "[Container]" &&
                        strings.HasPrefix(trimmed, "Volume=") &&
                        !strings.Contains(trimmed, "relabel=") &&
                        !strings.Contains(trimmed, ":Z") {

                        parts := strings.SplitN(trimmed, "=", 2)
                        if len(parts) == 2 {
                                volumeParts := strings.Split(parts[1], ":")
                                if len(volumeParts) >= 2 {
                                        newLine = fmt.Sprintf("Volume=%s:%s:relabel=private", volumeParts[0], volumeParts[1])
                                }
                        }
                }

                // 2. Port Intelligence: Offset ports < 1024 if rootless
                // Only applies inside [Container] section
                if c.IsRootless &&
                        currentSection == "[Container]" &&
                        strings.HasPrefix(trimmed, "Port=") {

                        parts := strings.SplitN(trimmed, "=", 2)
                        if len(parts) == 2 {
                                var port int
                                _, err := fmt.Sscanf(parts[1], "%d", &port)
                                if err == nil && port < 1024 {
                                        newLines = append(newLines, fmt.Sprintf("Port=%d", port+rootlessPortOffset))
                                        continue
                                }
                        }
                }

                newLines = append(newLines, newLine)
        }

        // 3. Systemd Optimizations

        // Add [Install] section for containers and networks
        if strings.HasSuffix(newPath, ".container") || strings.HasSuffix(newPath, ".network") {
                newLines = append(newLines, "", "[Install]", "WantedBy=default.target")
        }

        // Insert AutoUpdate=registry directly after [Container] header
        for i, line := range newLines {
                if strings.TrimSpace(line) == "[Container]" {
                        newLines = append(newLines[:i+1], append([]string{"AutoUpdate=registry"}, newLines[i+1:]...)...)
                        break
                }
        }

        output := strings.Join(newLines, "\n")
        return os.WriteFile(newPath, []byte(output), 0644)
}
