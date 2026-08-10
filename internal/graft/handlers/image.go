package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/preprocess"
)

func composePolicyToQuadlet(p string) string {
	switch strings.ToLower(p) {
	case "":
		return ""
	case "always":
		return "always"
	case "never":
		return "never"
	case "missing", "if_not_present":
		return "missing"
	default:
		logger.Warn(fmt.Sprintf("unsupported pull_policy %q for .image quadlet, omitting Policy=", p))
		return ""
	}
}

func extractServiceName(fileName, projectPrefix string) string {
	name := fileName
	if idx := strings.LastIndex(name, ".container"); idx >= 0 {
		name = name[:idx]
	}
	return strings.TrimPrefix(name, projectPrefix)
}

func makeImageFileName(containerPath string) string {
	dir := filepath.Dir(containerPath)
	base := filepath.Base(containerPath)
	return filepath.Join(dir, strings.TrimSuffix(base, ".container")+".image")
}

func imageBaseName(containerPath string) string {
	base := filepath.Base(containerPath)
	return strings.TrimSuffix(base, ".container") + ".image"
}

// ImageQuadletHandler creates .image quadlet files for every .container file
// in the cooked output map. It extracts Image= from the container, moves
// image-related fields into a new .image file, and updates the container
// to reference the .image quadlet.
func ImageQuadletHandler(
	files map[string]string,
	projectName string,
	services map[string]preprocess.ServiceImageSpec,
	buildSpecs map[string]*preprocess.ServiceBuildSpec,
) map[string]string {
	prefix := "cq-" + projectName + "-"

	for containerPath, containerContent := range files {
		if !strings.HasSuffix(containerPath, ".container") {
			continue
		}

		containerBase := filepath.Base(containerPath)
		serviceName := extractServiceName(containerBase, prefix)

		if _, isBuilt := buildSpecs[serviceName]; isBuilt {
			continue
		}

		spec, ok := services[serviceName]
		if !ok {
			logger.Warn(fmt.Sprintf("no service spec for %s, skipping .image generation", containerBase))
			continue
		}

		var imageValue string
		for _, line := range strings.Split(containerContent, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Image=") {
				imageValue = strings.TrimSpace(strings.TrimPrefix(trimmed, "Image="))
				break
			}
		}
		if imageValue == "" {
			continue
		}

		imageFileName := imageBaseName(containerPath)
		imageFilePath := makeImageFileName(containerPath)

		var imageLines []string
		imageLines = append(imageLines, "[Image]")
		imageLines = append(imageLines, "Image="+imageValue)

		if policy := composePolicyToQuadlet(spec.PullPolicy); policy != "" {
			imageLines = append(imageLines, "Policy="+policy)
		}
		if spec.OS != "" {
			imageLines = append(imageLines, "OS="+spec.OS)
		}
		if spec.Arch != "" {
			imageLines = append(imageLines, "Arch="+spec.Arch)
		}
		if spec.Variant != "" {
			imageLines = append(imageLines, "Variant="+spec.Variant)
		}

		imageLines = append(imageLines,
			"Retry=3",
			"RetryDelay=5s",
			"",
			"[Install]",
			"WantedBy=default.target",
		)

		imageContent := strings.Join(imageLines, "\n") + "\n"

		var containerLines []string
		for _, line := range strings.Split(containerContent, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Image=") {
				containerLines = append(containerLines, "Image="+imageFileName)
			} else {
				containerLines = append(containerLines, line)
			}
		}
		updatedContainer := strings.Join(containerLines, "\n")

		if err := os.WriteFile(imageFilePath, []byte(imageContent), 0644); err != nil {
			logger.Warn(fmt.Sprintf("failed to write .image file %s: %v", imageFilePath, err))
			continue
		}
		if err := os.WriteFile(containerPath, []byte(updatedContainer), 0644); err != nil {
			logger.Warn(fmt.Sprintf("failed to update .container file %s: %v", containerPath, err))
			continue
		}

		logger.Info(fmt.Sprintf("Created %s (Image=%s)", imageFileName, imageValue))
		logger.Info(fmt.Sprintf("Updated %s to reference %s", containerBase, imageFileName))

		files[imageFilePath] = imageContent
		files[containerPath] = updatedContainer
	}

	return files
}
