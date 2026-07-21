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
		cf.Services = make(map[string]map[string]interface{})
	}

	if cf.Networks == nil {
		cf.Networks = make(map[string]interface{})
	}

	// 1. Inject Container Names, Absolute-ize Paths & Inject Labels
	for serviceName, service := range cf.Services {
		if _, has := service["container_name"]; !has {
			service["container_name"] = fmt.Sprintf("%s-%s", e.ProjectName, serviceName)
			logger.Info(fmt.Sprintf("Injected container_name: %s-%s", e.ProjectName, serviceName))
		}

		// Normalize image names only for services without build config
		if img, ok := service["image"].(string); ok {
			if _, hasBuild := service["build"]; !hasBuild {
				originalImage := img
				service["image"] = normalizeImage(img)
				if service["image"] != originalImage {
					logger.Info(fmt.Sprintf("Normalized image: %s → %s", originalImage, service["image"]))
				}
			}
		}

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
		logger.Info("Created default network: cq-default")
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

	// 4. Replace build blocks with image directives so podlet never sees them.
	replaced := replaceBuildWithImage(&cf, e.ProjectName)
	if len(replaced) > 0 {
		logger.Info("Replaced build blocks with image directives for: " + strings.Join(replaced, ", "))
	}

	// 5. Marshal back to YAML
	output, err := yaml.Marshal(&cf)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal processed compose file: %w", err)
	}

	return output, nil
}

// replaceBuildWithImage mutates cf in-place: for every service that has a
// build: block it sets image to <project>-<service>:latest and deletes the
// build key. It returns the list of service names that were replaced.
func replaceBuildWithImage(cf *ComposeFile, projectName string) []string {
	var replaced []string
	for name := range cf.Services {
		if _, hasBuild := cf.Services[name]["build"]; !hasBuild {
			continue
		}
		tag := fmt.Sprintf("%s-%s:latest", projectName, name)
		cf.Services[name]["image"] = tag
		delete(cf.Services[name], "build")
		replaced = append(replaced, name)
	}
	return replaced
}

// GetBuildInfo returns build configuration for services that have it
func (e *Engine) GetBuildInfo(input []byte) (map[string]*BuildInfo, error) {
	var cf ComposeFile

	if err := yaml.Unmarshal(input, &cf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal compose file: %w", err)
	}

	buildInfo := make(map[string]*BuildInfo)

	for serviceName, service := range cf.Services {
		buildRaw, hasBuild := service["build"]
		if !hasBuild {
			continue
		}

		bc := &BuildConfig{}

		switch v := buildRaw.(type) {
		case string:
			bc.Context = v
			bc.Dockerfile = "Dockerfile"
		case map[string]interface{}:
			if ctx, ok := v["context"].(string); ok {
				bc.Context = ctx
			}
			if df, ok := v["dockerfile"].(string); ok {
				bc.Dockerfile = df
			}
			if target, ok := v["target"].(string); ok {
				bc.Target = target
			}
			if args, ok := v["args"].(map[string]interface{}); ok {
				bc.Args = make(map[string]string)
				for k, val := range args {
					bc.Args[k] = buildArgValue(val)
				}
			} else if argsList, ok := v["args"].([]interface{}); ok {
				bc.Args = make(map[string]string)
				for _, item := range argsList {
					if s, ok := item.(string); ok {
						if idx := strings.Index(s, "="); idx >= 0 {
							bc.Args[s[:idx]] = s[idx+1:]
						}
					}
				}
			}
		}

		context := bc.Context
		if context == "" {
			context = "."
		}

		// Resolve context to absolute path
		if !filepath.IsAbs(context) {
			context = filepath.Join(e.WorkingDirectory, context)
		}

		dockerfile := bc.Dockerfile
		if dockerfile == "" {
			dockerfile = "Dockerfile"
		}

		args := []string{}
		for k, v := range bc.Args {
			args = append(args, fmt.Sprintf("%s=%s", k, v))
		}

		buildInfo[serviceName] = &BuildInfo{
			Context:    context,
			Dockerfile: dockerfile,
			Args:       args,
			Target:     bc.Target,
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
