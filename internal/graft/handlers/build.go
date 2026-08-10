package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/preprocess"
)

func BuildQuadletHandler(
	files map[string]string,
	buildSpecs map[string]*preprocess.ServiceBuildSpec,
	projectName string,
) map[string]string {
	prefix := "cq-" + projectName + "-"

	for containerPath := range files {
		if !strings.HasSuffix(containerPath, ".container") {
			continue
		}

		containerBase := filepath.Base(containerPath)
		serviceName := strings.TrimSuffix(strings.TrimPrefix(containerBase, prefix), ".container")
		spec, ok := buildSpecs[serviceName]
		if !ok {
			continue
		}

		buildFileName := strings.TrimSuffix(containerBase, ".container") + ".build"
		buildPath := strings.TrimSuffix(containerPath, ".container") + ".build"

		var lines []string
		lines = append(lines, "[Build]")
		lines = append(lines, "ImageTag="+spec.ImageTag)
		lines = append(lines, "File="+spec.Dockerfile)
		lines = append(lines, "SetWorkingDirectory="+spec.Context)

		if spec.Target != "" {
			lines = append(lines, "Target="+spec.Target)
		}
		if spec.Network != "" {
			lines = append(lines, "Network="+spec.Network)
		}
		for k, v := range spec.Args {
			lines = append(lines, fmt.Sprintf("BuildArg=%s=%s", k, v))
		}
		for k, v := range spec.Labels {
			lines = append(lines, fmt.Sprintf("Label=%s=%s", k, v))
		}

		lines = append(lines, "", "[Install]", "WantedBy=default.target")

		content := strings.Join(lines, "\n") + "\n"

		if err := os.WriteFile(buildPath, []byte(content), 0644); err != nil {
			logger.Warn(fmt.Sprintf("failed to write .build file %s: %v", buildPath, err))
			continue
		}

		logger.Info(fmt.Sprintf("Created %s (ImageTag=%s)", buildFileName, spec.ImageTag))

		files[buildPath] = content

		containerContent := files[containerPath]
		var updatedLines []string
		for _, line := range strings.Split(containerContent, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Image=") {
				updatedLines = append(updatedLines, "Image="+buildFileName)
			} else if trimmed == "AutoUpdate=registry" {
				continue
			} else {
				updatedLines = append(updatedLines, line)
			}
		}
		updatedContent := strings.Join(updatedLines, "\n")

		files[containerPath] = updatedContent
		if err := os.WriteFile(containerPath, []byte(updatedContent), 0644); err != nil {
			logger.Warn(fmt.Sprintf("failed to update .container file %s: %v", containerPath, err))
		}
		logger.Info(fmt.Sprintf("Updated %s Image=%s", containerBase, buildFileName))
	}

	return files
}
