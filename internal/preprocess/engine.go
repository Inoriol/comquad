package preprocess

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/logger"
	"gopkg.in/yaml.v3"
)

// Engine handles the pre-processing of Compose files
type Engine struct {
	ProjectName      string
	WorkingDirectory string
}

// NewEngine creates a new pre-processing engine
func NewEngine(projectName string, workingDir string) *Engine {
	return &Engine{
		ProjectName:      projectName,
		WorkingDirectory: workingDir,
	}
}

// Process takes a raw YAML input and applies normalization rules.
func (e *Engine) Process(input []byte) ([]byte, error) {
	var cf ComposeFile

	if err := yaml.Unmarshal(input, &cf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal compose file: %w", err)
	}

	if cf.Services == nil {
		cf.Services = make(map[string]map[string]interface{})
	}

	if cf.Networks == nil {
		cf.Networks = make(map[string]interface{})
	}

	// 1. Inject Container Names, Absolute-ize Paths & Inject Labels
	for serviceName, service := range cf.Services {
		if _, hasBuild := service["build"]; hasBuild {
			return nil, fmt.Errorf("service %q uses a build: block — builds are not supported yet", serviceName)
		}
		if _, has := service["container_name"]; !has {
			service["container_name"] = fmt.Sprintf("%s-%s", e.ProjectName, serviceName)
			logger.Info(fmt.Sprintf("Injected container_name: %s-%s", e.ProjectName, serviceName))
		}

		if img, ok := service["image"].(string); ok {
			originalImage := img
			service["image"] = normalizeImage(img)
			if service["image"] != originalImage {
				logger.Info(fmt.Sprintf("Normalized image: %s → %s", originalImage, service["image"]))
			}
		}

		delete(service, "pull_policy")
		delete(service, "platform")

		// Absolute-ize volumes
		if vols, ok := service["volumes"].([]interface{}); ok {
			for i, volInterface := range vols {
				vol, ok := volInterface.(string)
				if !ok {
					continue
				}
				if strings.HasPrefix(vol, "./") || strings.HasPrefix(vol, "..") {
					parts := strings.Split(vol, ":")
					absPath, err := filepath.Abs(filepath.Join(e.WorkingDirectory, parts[0]))
					if err != nil {
						return nil, fmt.Errorf("failed to resolve path for volume %s: %w", vol, err)
					}
					if len(parts) > 1 {
						vols[i] = fmt.Sprintf("%s:%s", absPath, strings.Join(parts[1:], ":"))
					} else {
						vols[i] = absPath
					}
					logger.Info(fmt.Sprintf("Normalized volume path: %s → %s", vol, vols[i]))
				}
			}
		}
	}

	// 2. Automatic Networking: Ensure a default bridge network exists.
	// Only inject cq-default when the compose file defines no networks at all.
	// Services are only auto-attached to cq-default if it was actually injected,
	// preventing dangling network references when user-defined networks exist.
	defaultNetworkInjected := false
	if len(cf.Networks) == 0 {
		cf.Networks["cq-default"] = map[string]interface{}{
			"driver": "bridge",
		}
		defaultNetworkInjected = true
		logger.Action("Created default network: cq-default")
	}

	// Ensure all services are attached to at least one network.
	// Only attach to cq-default if we just created it.
	if defaultNetworkInjected {
		for serviceName, service := range cf.Services {
			if _, hasNetworks := service["networks"]; !hasNetworks {
				service["networks"] = []string{"cq-default"}
				logger.Info(fmt.Sprintf("Auto-attached '%s' to network 'cq-default'", serviceName))
			}
		}
	}

	// 3. Inject force-volume labels into top-level named volumes to ensure podlet generates .volume files.
	for name, vol := range cf.Volumes {
		if vol == nil {
			cf.Volumes[name] = make(map[string]interface{})
			vol = cf.Volumes[name]
		}
		switch labels := vol["labels"].(type) {
		case StringMap:
			labels["com.comquad.force-volume"] = "true"
			vol["labels"] = labels
		case map[string]interface{}:
			sm := make(StringMap)
			for k, v := range labels {
				if s, ok := v.(string); ok {
					sm[k] = s
				}
			}
			sm["com.comquad.force-volume"] = "true"
			vol["labels"] = sm
		case []interface{}:
			// Convert list format to StringMap, preserving existing entries
			sm := make(StringMap)
			for _, item := range labels {
				if s, ok := item.(string); ok {
					if idx := strings.Index(s, "="); idx >= 0 {
						sm[s[:idx]] = s[idx+1:]
					}
				}
			}
			sm["com.comquad.force-volume"] = "true"
			vol["labels"] = sm
		default:
			vol["labels"] = StringMap{"com.comquad.force-volume": "true"}
		}
	}

	// 4. Marshal back to YAML
	output, err := yaml.Marshal(&cf)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal processed compose file: %w", err)
	}

	return output, nil
}

// normalizeImage ensures the image has a full registry path.
// Images without a registry (e.g. "nginx", "postgres:15") get docker.io/library/ prepended.
// Images from custom registries (e.g. "myregistry.com/image") are left unchanged.
// Images already prefixed with docker.io/ are left unchanged.
func normalizeImage(image string) string {
	// Already fully qualified
	if strings.Contains(image, "/") {
		parts := strings.SplitN(image, "/", 2)
		first := parts[0]
		// Already docker.io
		if first == "docker.io" {
			return image
		}
		// Looks like a registry (contains . or :port in the first segment)
		if strings.Contains(first, ".") || isRegistryWithPort(first) {
			return image
		}
		// e.g. "library/nginx" — no registry, treat as docker hub
		return "docker.io/" + image
	}

	// No slash at all — bare image name, default to docker hub
	return "docker.io/library/" + image
}

// isRegistryWithPort checks if s looks like a registry hostname with a port number.
// A registry with port has the form "hostname:port" where port is all digits.
// This distinguishes "localhost:5000" (registry) from "myapp:v1" (image with tag).
func isRegistryWithPort(s string) bool {
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		port := s[idx+1:]
		if len(port) > 0 && len(port) <= 5 {
			for _, c := range port {
				if c < '0' || c > '9' {
					return false
				}
			}
			return true
		}
	}
	return false
}

// ExtractServiceImageSpecs parses raw compose YAML and returns per-service
// image metadata before any preprocessing has been applied. This allows the
// graft step to populate .image quadlet files while the preprocessor strips
// pull_policy and platform fields that podlet does not understand.
func ExtractServiceImageSpecs(composeData []byte) (map[string]ServiceImageSpec, error) {
	var cf ComposeFile
	if err := yaml.Unmarshal(composeData, &cf); err != nil {
		return nil, fmt.Errorf("failed to parse compose file for image specs: %w", err)
	}

	specs := make(map[string]ServiceImageSpec, len(cf.Services))
	for svcName, svc := range cf.Services {
		spec := ServiceImageSpec{ServiceName: svcName}

		if img, ok := svc["image"].(string); ok && img != "" {
			spec.Image = normalizeImage(img)
		}

		if pp, ok := svc["pull_policy"].(string); ok {
			spec.PullPolicy = pp
		}

		if plat, ok := svc["platform"].(string); ok {
			parts := strings.SplitN(plat, "/", 2)
			spec.OS = parts[0]
			if len(parts) > 1 {
				archVariant := parts[1]
				if idx := strings.Index(archVariant, "/"); idx >= 0 {
					spec.Arch = archVariant[:idx]
					spec.Variant = archVariant[idx+1:]
				} else {
					spec.Arch = archVariant
				}
			}
		}

		specs[svcName] = spec
	}
	return specs, nil
}
