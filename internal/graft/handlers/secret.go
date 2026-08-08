package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/preprocess"
)

func secretLookupName(def preprocess.SecretDef) string {
	if def.ExternalName != "" {
		return def.ExternalName
	}
	return def.Name
}

func insertContainerDirective(lines []string, directive string) []string {
	containerIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "[Container]" {
			containerIdx = i
			break
		}
	}
	if containerIdx < 0 {
		return lines
	}

	insertAt := containerIdx + 1
	for i := containerIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" && strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			break
		}
		if trimmed != "" {
			insertAt = i + 1
		}
	}

	result := make([]string, 0, len(lines)+2)
	result = append(result, lines[:insertAt]...)
	result = append(result, directive)
	result = append(result, lines[insertAt:]...)
	return result
}

func addSELinuxToVolume(directive string) string {
	value := strings.TrimPrefix(directive, "Volume=")
	parts := strings.SplitN(value, ":", 3)

	if len(parts) <= 2 {
		return directive + ":z"
	}

	options := strings.Split(parts[2], ",")
	for _, opt := range options {
		if opt == "z" || opt == "Z" {
			return directive
		}
	}
	return "Volume=" + parts[0] + ":" + parts[1] + ":" + parts[2] + ",z"
}

// SecretHandler processes compose secrets and injects the corresponding
// quadlet directives into .container files.
func SecretHandler(
	files map[string]string,
	projectName string,
	secretDefs map[string]preprocess.SecretDef,
	serviceRefs preprocess.ServiceSecretRefs,
	secretsDir string,
	dryRun bool,
	selinuxEnabled bool,
) map[string]string {
	prefix := "cq-" + projectName + "-"

	for containerPath, containerContent := range files {
		if !strings.HasSuffix(containerPath, ".container") {
			continue
		}

		containerBase := filepath.Base(containerPath)
		serviceName := extractServiceName(containerBase, prefix)
		refs, ok := serviceRefs[serviceName]
		if !ok || len(refs) == 0 {
			continue
		}

		lines := strings.Split(containerContent, "\n")

		for _, ref := range refs {
			def, ok := secretDefs[ref.Source]
			if !ok {
				logger.Warn(fmt.Sprintf("secret %q referenced by service %q not found in definitions", ref.Source, serviceName))
				continue
			}

			if def.External {
				lines = insertContainerDirective(lines, "Secret="+secretLookupName(def))
				logger.Info(fmt.Sprintf("Injected Secret=%s in %s", secretLookupName(def), containerBase))
				continue
			}

			var sourcePath string

			if def.File != "" {
				sourcePath = def.File
			} else if def.Environment != "" {
				sourcePath = filepath.Join(secretsDir, ref.Source)
				if !dryRun {
					val := def.Content
					if val == "" {
						val = os.Getenv(def.Environment)
					}
					if val == "" {
						logger.Warn(fmt.Sprintf("secret %q: environment variable %q is not set or empty", def.Name, def.Environment))
					}
					if err := os.MkdirAll(secretsDir, 0700); err != nil {
						logger.Warn(fmt.Sprintf("failed to create secrets directory %s: %v", secretsDir, err))
					} else if err := os.WriteFile(sourcePath, []byte(val), 0600); err != nil {
						logger.Warn(fmt.Sprintf("failed to write secret %s: %v", ref.Source, err))
					}
				}
			}

			if sourcePath == "" {
				logger.Warn(fmt.Sprintf("secret %q has no source (file or environment)", def.Name))
				continue
			}

			target := ref.Target
			if target == "" {
				target = "/run/secrets/" + ref.Source
			}

			volumeDirective := "Volume=" + sourcePath + ":" + target + ":ro"
			if selinuxEnabled {
				volumeDirective = addSELinuxToVolume(volumeDirective)
			}
			lines = insertContainerDirective(lines, volumeDirective)

			logger.Info(fmt.Sprintf("Injected %s in %s", volumeDirective, containerBase))
		}

		updatedContent := strings.Join(lines, "\n")

		if !dryRun {
			if err := os.WriteFile(containerPath, []byte(updatedContent), 0644); err != nil {
				logger.Warn(fmt.Sprintf("failed to update .container file %s: %v", containerPath, err))
			}
		}

		files[containerPath] = updatedContent
	}

	return files
}
