package preprocess

import (
	"fmt"
	"path/filepath"
	"strings"

	"comquad/internal/logger"
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
// It also extracts build configuration from services.
func (e *Engine) Process(input []byte) ([]byte, error) {
	var cf ComposeFile

	if err := yaml.Unmarshal(input, &cf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal compose file: %w", err)
	}

	if cf.Services == nil {
		cf.Services = make(map[string]*Service)
	}

	if cf.Networks == nil {
		cf.Networks = make(map[string]*Network)
	}

	// 1. Inject Container Names, Absolute-ize Paths & Inject Labels
	for serviceName, service := range cf.Services {
		if service.ContainerName == "" {
			service.ContainerName = fmt.Sprintf("%s-%s", e.ProjectName, serviceName)
			logger.Info(fmt.Sprintf("Injected container_name: %s-%s", e.ProjectName, serviceName))
		}

		// Normalize image names only for services without build config
		if service.Image != "" && service.Build == nil {
			originalImage := service.Image
			service.Image = normalizeImage(service.Image)
			if service.Image != originalImage {
				logger.Info(fmt.Sprintf("Normalized image: %s → %s", originalImage, service.Image))
			}
		}

		// Absolute-ize volumes
		for i, vol := range service.Volumes {
			if strings.HasPrefix(vol, "./") || strings.HasPrefix(vol, "..") {
				parts := strings.Split(vol, ":")

				absPath, err := filepath.Abs(filepath.Join(e.WorkingDirectory, parts[0]))
				if err != nil {
					return nil, fmt.Errorf("failed to resolve path for volume %s: %w", vol, err)
				}

				// Rejoin from parts[1:] to preserve all options (e.g. :ro, :z, :cached)
				if len(parts) > 1 {
					service.Volumes[i] = fmt.Sprintf("%s:%s", absPath, strings.Join(parts[1:], ":"))
				} else {
					service.Volumes[i] = absPath
				}
				logger.Info(fmt.Sprintf("Normalized volume path: %s → %s", vol, service.Volumes[i]))
			}
		}
	}

	// 2. Automatic Networking: Ensure a default bridge network exists.
	// Only inject cq-default when the compose file defines no networks at all.
	// Services are only auto-attached to cq-default if it was actually injected,
	// preventing dangling network references when user-defined networks exist.
	defaultNetworkInjected := false
	if len(cf.Networks) == 0 {
		cf.Networks["cq-default"] = &Network{
			Driver: "bridge",
		}
		defaultNetworkInjected = true
		logger.Info("Created default network: cq-default")
	}

	// Ensure all services are attached to at least one network.
	// Only attach to cq-default if we just created it.
	if defaultNetworkInjected {
		for serviceName, service := range cf.Services {
			if len(service.Networks) == 0 {
				service.Networks = append(service.Networks, "cq-default")
				logger.Info(fmt.Sprintf("Auto-attached '%s' to network 'cq-default'", serviceName))
			}
		}
	}

	// 3. Inject force-volume labels into top-level named volumes to ensure podlet generates .volume files.
	for name, vol := range cf.Volumes {
		if vol == nil {
			cf.Volumes[name] = &Volume{}
			vol = cf.Volumes[name]
		}
		if vol.Labels == nil {
			vol.Labels = make(map[string]string)
		}
		vol.Labels["com.comquad.force-volume"] = "true"
	}

	// 4. Marshal back to YAML
	output, err := yaml.Marshal(&cf)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal processed compose file: %w", err)
	}

	return output, nil
}

// GetBuildInfo returns build configuration for services that have it
func (e *Engine) GetBuildInfo(input []byte) (map[string]*BuildInfo, error) {
	var cf ComposeFile

	if err := yaml.Unmarshal(input, &cf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal compose file: %w", err)
	}

	buildInfo := make(map[string]*BuildInfo)

	for serviceName, service := range cf.Services {
		if service.Build == nil {
			continue
		}

		context := service.Build.Context
		if context == "" {
			context = "."
		}

		// Resolve context to absolute path
		if !filepath.IsAbs(context) {
			context = filepath.Join(e.WorkingDirectory, context)
		}

		dockerfile := service.Build.Dockerfile
		if dockerfile == "" {
			dockerfile = "Dockerfile"
		}

		args := []string{}
		for k, v := range service.Build.Args {
			args = append(args, fmt.Sprintf("%s=%s", k, v))
		}

		buildInfo[serviceName] = &BuildInfo{
			Context:    context,
			Dockerfile: dockerfile,
			Args:       args,
			Target:     service.Build.Target,
			Service:    serviceName,
		}
	}

	return buildInfo, nil
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
		// Looks like a registry (contains . or : in the first segment)
		if strings.Contains(first, ".") || strings.Contains(first, ":") {
			return image
		}
		// e.g. "library/nginx" — no registry, treat as docker hub
		return "docker.io/" + image
	}

	// No slash at all — bare image name, default to docker hub
	return "docker.io/library/" + image
}
