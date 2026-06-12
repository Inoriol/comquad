package orchestrator

import (
	"path/filepath"
	"strings"

	"comquad/internal/deploy"
)

// MatchContainer finds a container quadlet file matching the given arg.
func MatchContainer(projectName string, state deploy.ProjectState, arg string) string {
	servicePrefix := "cq-" + projectName + "-"

	for _, f := range state.Files {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		base := filepath.Base(f)
		nameWithoutExt := strings.TrimSuffix(base, ".container")

		if base == arg {
			return f
		}
		if nameWithoutExt == arg {
			return f
		}
		if strings.TrimSuffix(arg, ".service") == nameWithoutExt {
			return f
		}
		if strings.TrimPrefix(nameWithoutExt, servicePrefix) == arg {
			return f
		}
	}
	return ""
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
