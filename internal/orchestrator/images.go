package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"comquad/internal/build"
	"comquad/internal/logger"
	"comquad/internal/preprocess"
)

// handleImages builds or pulls images based on the compose file and strategy.
func (o *Orchestrator) handleImages(projectFiles []string, buildInfo map[string]*preprocess.BuildInfo, forceBuild bool, pullStrategy string) error {
	sortedServiceNames := make([]string, 0, len(buildInfo))
	for name := range buildInfo {
		sortedServiceNames = append(sortedServiceNames, name)
	}
	sort.Strings(sortedServiceNames)

	for _, serviceName := range sortedServiceNames {
		info := buildInfo[serviceName]
		imageTag := build.GenerateBuildTag(o.projectName, serviceName)

		shouldBuild := forceBuild
		if !shouldBuild {
			engine := &build.Engine{}
			shouldBuild = !engine.ImageExists(imageTag)
		}

		if shouldBuild {
			engine := &build.Engine{}
			if err := engine.BuildService(
				serviceName,
				info.Context,
				info.Dockerfile,
				info.Args,
				info.Target,
				imageTag,
			); err != nil {
				return fmt.Errorf("failed to build image for service %s: %w", serviceName, err)
			}
			logger.Success("Built image: " + imageTag)
		} else {
			logger.Action("Image already exists locally, skipping build: " + imageTag)
		}
	}

	imagePullStrategy, err := build.ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", f, err)
		}

		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Image=") {
				image := strings.TrimSpace(strings.TrimPrefix(line, "Image="))

				if o.isBuildGeneratedImage(image, buildInfo) {
					continue
				}

				engine := &build.Engine{
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

// printDryRun prints a preview of what `comquad up` would deploy without
// writing anything to the real systemd directory or starting any units.
func (o *Orchestrator) printDryRun(
	projectFiles []string,
	previewDir string,
	targetDir string,
	buildInfo map[string]*preprocess.BuildInfo,
	pullStrategy string,
) error {
	logger.Printf("Dry run — project: %s\n", o.projectName)
	logger.Printf("Target directory: %s\n\n", targetDir)

	imagePullStrategy, err := build.ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	sortedNames := make([]string, 0, len(buildInfo))
	for name := range buildInfo {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, serviceName := range sortedNames {
		info := buildInfo[serviceName]
		imageTag := build.GenerateBuildTag(o.projectName, serviceName)
		engine := &build.Engine{}
		if engine.ImageExists(imageTag) {
			logger.Printf("[image] %-12s %s  (already exists locally, would skip build)\n", serviceName, imageTag)
		} else {
			logger.Printf("[image] %-12s %s  (would build from %s)\n", serviceName, imageTag, info.Context)
		}
	}

	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read preview file %s: %w", f, err)
		}
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "Image=") {
				continue
			}
			image := strings.TrimSpace(strings.TrimPrefix(line, "Image="))
			if o.isBuildGeneratedImage(image, buildInfo) {
				break
			}
			switch imagePullStrategy {
			case build.PullAlways:
				logger.Printf("[image] %-12s %s  (would pull: always)\n", filepath.Base(f), image)
			case build.PullMissing:
				engine := &build.Engine{}
				if engine.ImageExists(image) {
					logger.Printf("[image] %-12s %s  (already exists locally, would skip pull)\n", filepath.Base(f), image)
				} else {
					logger.Printf("[image] %-12s %s  (would pull: not found locally)\n", filepath.Base(f), image)
				}
			case build.PullNever:
				logger.Printf("[image] %-12s %s  (pull skipped: never)\n", filepath.Base(f), image)
			}
			break
		}
	}

	if len(buildInfo) > 0 || len(projectFiles) > 0 {
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

		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read preview file %s: %w", f, err)
		}

		logger.Print(separator)
		logger.Printf("  %s\n", targetPath)
		logger.Print(separator)
		logger.Print(strings.TrimRight(string(content), "\n"))
		logger.Print("")
	}

	logger.Print("Dry run complete — nothing was written, no units started.")
	return nil
}
