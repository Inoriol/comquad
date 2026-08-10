package orchestrator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/graft"
	"github.com/Inoriol/comquad/internal/logger"
)

func resolveContainerImage(imageValue string, fileContents map[string]string) string {
	if !strings.HasSuffix(imageValue, ".image") {
		return imageValue
	}

	for path, content := range fileContents {
		if filepath.Base(path) == imageValue {
			for _, line := range strings.Split(content, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "Image=") {
					return strings.TrimPrefix(trimmed, "Image=")
				}
			}
		}
	}

	return imageValue
}

// handleImages pulls images based on the pull strategy.
func (o *Orchestrator) handleImages(projectFiles []string, fileContents map[string]string, pullStrategy string) error {
	imagePullStrategy, err := graft.ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}

		if hasBuildFile(projectFiles, f) {
			logger.Action("Skipping pull for " + filepath.Base(f) + " (built from .build quadlet)")
			continue
		}

		content, ok := fileContents[f]
		if !ok {
			return fmt.Errorf("content not found for %s", f)
		}

		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Image=") {
				image := strings.TrimSpace(strings.TrimPrefix(line, "Image="))

				image = resolveContainerImage(image, fileContents)

				engine := &graft.Engine{
					PullStrategy: imagePullStrategy,
				}

				if err := engine.HandleImage("", image); err != nil {
					return fmt.Errorf("failed to handle image %s: %w", image, err)
				}
				logger.Success("Handled image: " + image)
				break
			}
		}
	}

	return nil
}

func hasBuildFile(projectFiles []string, containerPath string) bool {
	containerBase := filepath.Base(containerPath)
	buildBase := strings.TrimSuffix(containerBase, ".container") + ".build"
	for _, f := range projectFiles {
		if filepath.Base(f) == buildBase {
			return true
		}
	}
	return false
}

// printDryRun prints a preview of what `comquad up` would deploy without
// writing anything to the real systemd directory or starting any units.
func (o *Orchestrator) printDryRun(
	projectFiles []string,
	fileContents map[string]string,
	previewDir string,
	targetDir string,
	pullStrategy string,
) error {
	logger.Printf("Dry run — project: %s\n", o.projectName)
	logger.Printf("Target directory: %s\n\n", targetDir)

	imagePullStrategy, err := graft.ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		content, ok := fileContents[f]
		if !ok {
			return fmt.Errorf("content not found for %s", f)
		}
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "Image=") {
				continue
			}
			image := strings.TrimSpace(strings.TrimPrefix(line, "Image="))

			image = resolveContainerImage(image, fileContents)

			if hasBuildFile(projectFiles, f) {
				logger.Printf("[build] %-12s %s  (would be built locally, no pull)\n", filepath.Base(f), image)
				break
			}

			switch imagePullStrategy {
			case graft.PullAlways:
				logger.Printf("[image] %-12s %s  (would pull: always)\n", filepath.Base(f), image)
			case graft.PullMissing:
				engine := &graft.Engine{}
				if engine.ImageExists(image) {
					logger.Printf("[image] %-12s %s  (already exists locally, would skip pull)\n", filepath.Base(f), image)
				} else {
					logger.Printf("[image] %-12s %s  (would pull: not found locally)\n", filepath.Base(f), image)
				}
			case graft.PullNever:
				logger.Printf("[image] %-12s %s  (pull skipped: never)\n", filepath.Base(f), image)
			}
			break
		}
	}

	if len(projectFiles) > 0 {
		logger.Print("")
	}

	logger.Printf("%d quadlet file(s) would be written:\n\n", len(projectFiles))
	separator := strings.Repeat("─", 60)

	for _, f := range projectFiles {
		rel, err := filepath.Rel(previewDir, f)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}
		targetPath := filepath.Join(targetDir, rel)

		content, ok := fileContents[f]
		if !ok {
			return fmt.Errorf("content not found for %s", f)
		}

		logger.Print(separator)
		logger.Printf("  %s\n", targetPath)
		logger.Print(separator)
		logger.Print(strings.TrimRight(content, "\n"))
		logger.Print("")
	}

	logger.Print("Dry run complete — nothing was written, no units started.")
	return nil
}
