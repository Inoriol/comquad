package preprocess

import (
	"fmt"
	"path/filepath"
	"strings"

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

// Process takes a raw YAML input and applies normalization rules
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
		}

		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}

		service.Labels["com.comquad.project"] = e.ProjectName

		// Normalize image names to full Docker Hub path when no registry is specified
		if service.Image != "" {
			service.Image = normalizeImage(service.Image)
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
			}
		}
	}

	// 2. Automatic Networking: Ensure a default bridge network exists
	if len(cf.Networks) == 0 {
		cf.Networks["cq-default"] = &Network{
			Driver: "bridge",
		}
	}

	// Ensure all services are attached to at least one network
	for _, service := range cf.Services {
		if len(service.Networks) == 0 {
			service.Networks = append(service.Networks, "cq-default")
		}
	}

	// 3. Marshal back to YAML
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
