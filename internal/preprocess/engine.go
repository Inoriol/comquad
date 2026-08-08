package preprocess

import (
	"bufio"
	"fmt"
	"os"
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

	// 1. Strip secrets from services (they are handled by the graft step)
	for _, service := range cf.Services {
		delete(service, "secrets")
	}

	// 2. Inject Container Names, Absolute-ize Paths & Inject Labels
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

	// 3. Automatic Networking: Ensure a default bridge network exists.
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

	// 4. Inject force-volume labels into top-level named volumes to ensure podlet generates .volume files.
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

	// 5. Strip top-level secrets and marshal back to YAML
	cf.Secrets = nil

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

// ExtractSecretSpecs parses raw compose YAML and extracts secret definitions
// and per-service secret references before any preprocessing has been applied.
func ExtractSecretSpecs(composeData []byte, workingDir string) (map[string]SecretDef, ServiceSecretRefs, error) {
	var cf ComposeFile
	if err := yaml.Unmarshal(composeData, &cf); err != nil {
		return nil, nil, fmt.Errorf("failed to parse compose file for secret specs: %w", err)
	}

	secretDefs := make(map[string]SecretDef, len(cf.Secrets))
	for name, raw := range cf.Secrets {
		def := SecretDef{Name: name}

		if ext, ok := raw["external"]; ok {
			if extBool, isBool := ext.(bool); isBool && extBool {
				def.External = true
			}
		}

		if nameVal, ok := raw["name"].(string); ok {
			def.ExternalName = nameVal
		}

		if def.External {
			for key := range raw {
				if key != "external" && key != "name" {
					return nil, nil, fmt.Errorf("secret %q: external=true requires no other attributes besides 'name' (found %q)", name, key)
				}
			}
		}

		if fileVal, ok := raw["file"].(string); ok && fileVal != "" {
			absPath, err := filepath.Abs(filepath.Join(workingDir, fileVal))
			if err != nil {
				return nil, nil, fmt.Errorf("secret %q: failed to resolve file path %q: %w", name, fileVal, err)
			}
			def.File = absPath
		}

		if envVal, ok := raw["environment"].(string); ok && envVal != "" {
			def.Environment = envVal
			def.Content = resolveEnvironmentSecret(envVal, workingDir)
		}

		secretDefs[name] = def
	}

	serviceRefs := make(ServiceSecretRefs, len(cf.Services))
	for svcName, svc := range cf.Services {
		rawSecrets, hasSecrets := svc["secrets"]
		if !hasSecrets {
			continue
		}

		var refs []SecretRef

		switch s := rawSecrets.(type) {
		case []interface{}:
			for _, item := range s {
				switch v := item.(type) {
				case string:
					refs = append(refs, SecretRef{Source: v})
				case map[string]interface{}:
					ref := SecretRef{}
					if src, ok := v["source"].(string); ok {
						ref.Source = src
					}
					if tgt, ok := v["target"].(string); ok {
						ref.Target = tgt
					}
					if ref.Source != "" {
						refs = append(refs, ref)
					}
				}
			}
		}

		if len(refs) > 0 {
			serviceRefs[svcName] = refs
		}
	}

	for svcName, refs := range serviceRefs {
		for _, ref := range refs {
			if _, ok := secretDefs[ref.Source]; !ok {
				return nil, nil, fmt.Errorf("service %q references undefined secret %q", svcName, ref.Source)
			}
		}
	}

	return secretDefs, serviceRefs, nil
}

func resolveEnvironmentSecret(varName, workingDir string) string {
	if val := os.Getenv(varName); val != "" {
		return val
	}

	dotEnv := loadDotEnv(filepath.Join(workingDir, ".env"))
	if val, ok := dotEnv[varName]; ok {
		return val
	}

	return ""
}

func loadDotEnv(path string) map[string]string {
	env := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return env
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx >= 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[0] == val[len(val)-1] {
				val = val[1 : len(val)-1]
			}
			env[key] = val
		}
	}
	return env
}
