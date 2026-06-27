package orchestrator

import (
	"os"
	"path/filepath"
	"strings"

	"comquad/internal/deploy"
)

// readContainerName reads the ContainerName= value from a .container quadlet file.
// Returns empty string if not found or if the file cannot be read.
func readContainerName(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ContainerName=") {
			return strings.TrimPrefix(line, "ContainerName=")
		}
	}
	return ""
}

// matchAllContainers returns all container quadlet files from state that match arg.
// Six patterns are tried in order: exact base name, name without extension,
// name without .service suffix, short name (strip cq-<project>- prefix),
// internal Podman name (strip cq- prefix), ContainerName= directive in the unit file.
func matchAllContainers(projectName string, state deploy.ProjectState, arg string) []string {
	servicePrefix := "cq-" + projectName + "-"
	var matches []string

	for _, f := range state.Files {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		base := filepath.Base(f)
		nameWithoutExt := strings.TrimSuffix(base, ".container")

		if base == arg ||
			nameWithoutExt == arg ||
			strings.TrimSuffix(arg, ".service") == nameWithoutExt ||
			strings.TrimPrefix(nameWithoutExt, servicePrefix) == arg ||
			strings.TrimPrefix(nameWithoutExt, "cq-") == arg {
			matches = append(matches, f)
			continue
		}

		// Sixth pattern: ContainerName= directive in the unit file
		if cn := readContainerName(f); cn == arg {
			matches = append(matches, f)
		}
	}
	return matches
}

// MatchContainer finds the first container quadlet file matching the given arg.
func MatchContainer(projectName string, state deploy.ProjectState, arg string) string {
	matches := matchAllContainers(projectName, state, arg)
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// MatchContainers finds all container quadlet files matching the given arg.
func MatchContainers(projectName string, state deploy.ProjectState, arg string) []string {
	return matchAllContainers(projectName, state, arg)
}

// MatchNetworkOrVolume finds a network or volume quadlet file matching the given arg.
func MatchNetworkOrVolume(projectName string, state deploy.ProjectState, arg string) string {
	for _, f := range state.Files {
		if !strings.HasSuffix(f, ".network") && !strings.HasSuffix(f, ".volume") {
			continue
		}
		base := filepath.Base(f)
		nameWithoutExt := strings.TrimSuffix(base, ".network")
		nameWithoutExt = strings.TrimSuffix(nameWithoutExt, ".volume")

		serviceName := nameWithoutExt
		if strings.HasSuffix(base, ".network") {
			serviceName = nameWithoutExt + "-network"
		} else if strings.HasSuffix(base, ".volume") {
			serviceName = nameWithoutExt + "-volume"
		}

		if base == arg {
			return f
		}
		if serviceName == arg {
			return f
		}
		if strings.TrimSuffix(arg, ".service") == serviceName {
			return f
		}
	}
	return ""
}
