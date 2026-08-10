package preprocess

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type resolvedBuild struct {
	Context    string
	Dockerfile string
	Extra      map[string]interface{}
}

func resolveDefaultDockerfile(ctx string) string {
	if _, err := os.Stat(filepath.Join(ctx, "Containerfile")); err == nil {
		return "Containerfile"
	}
	return "Dockerfile"
}

func resolveBuildPath(buildRaw interface{}, workingDir string) (*resolvedBuild, error) {
	switch v := buildRaw.(type) {
	case string:
		ctx, err := filepath.Abs(filepath.Join(workingDir, v))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve build context %q: %w", v, err)
		}
		return &resolvedBuild{
			Context:    ctx,
			Dockerfile: filepath.Join(ctx, resolveDefaultDockerfile(ctx)),
		}, nil
	case map[string]interface{}:
		ctx := "."
		if ctxRaw, ok := v["context"]; ok {
			if ctxStr, ok := ctxRaw.(string); ok {
				ctx = ctxStr
			}
		}
		absCtx, err := filepath.Abs(filepath.Join(workingDir, ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve context %q: %w", ctx, err)
		}

		dockerfile := ""
		if dfRaw, ok := v["dockerfile"]; ok {
			if dfStr, ok := dfRaw.(string); ok {
				dockerfile = dfStr
			}
		}
		if dockerfile == "" {
			dockerfile = resolveDefaultDockerfile(absCtx)
		}
		absDf := filepath.Join(absCtx, dockerfile)

		extra := make(map[string]interface{})
		for k, val := range v {
			if k != "context" && k != "dockerfile" {
				extra[k] = val
			}
		}

		return &resolvedBuild{
			Context:    absCtx,
			Dockerfile: absDf,
			Extra:      extra,
		}, nil
	default:
		return nil, fmt.Errorf("build field must be a string or a mapping, got %T", buildRaw)
	}
}

func extractBuildSpec(svcName string, buildRaw interface{}, workingDir, cacheDir, projectName string) (*ServiceBuildSpec, error) {
	resolved, err := resolveBuildPath(buildRaw, workingDir)
	if err != nil {
		return nil, fmt.Errorf("service %q: %w", svcName, err)
	}

	dockerfileContent, err := os.ReadFile(resolved.Dockerfile)
	if err != nil {
		return nil, fmt.Errorf("service %q: failed to read Dockerfile %s: %w", svcName, resolved.Dockerfile, err)
	}

	patchedContent := PatchDockerfile(dockerfileContent)

	patchedPath := filepath.Join(cacheDir, svcName+".Dockerfile")
	if err := os.MkdirAll(filepath.Dir(patchedPath), 0755); err != nil {
		return nil, fmt.Errorf("service %q: failed to create build cache dir: %w", svcName, err)
	}
	if err := os.WriteFile(patchedPath, patchedContent, 0644); err != nil {
		return nil, fmt.Errorf("service %q: failed to write patched Dockerfile: %w", svcName, err)
	}

	spec := &ServiceBuildSpec{
		ServiceName: svcName,
		HasBuild:    true,
		ImageTag:    projectName + "-" + svcName + ":latest",
		Context:     resolved.Context,
		Dockerfile:  patchedPath,
	}

	if target, ok := resolved.Extra["target"].(string); ok {
		spec.Target = target
	}
	if network, ok := resolved.Extra["network"].(string); ok {
		spec.Network = network
	}
	if noCache, ok := resolved.Extra["no_cache"].(bool); ok {
		spec.NoCache = noCache
	}
	if args, ok := resolved.Extra["args"]; ok {
		spec.Args = extractBuildArgs(args)
	}
	if labels, ok := resolved.Extra["labels"]; ok {
		spec.Labels = extractBuildLabels(labels)
	}
	if pull, ok := resolved.Extra["pull"].(string); ok {
		spec.PullPolicy = pull
	}

	spec.Extra = resolved.Extra

	return spec, nil
}

func extractBuildArgs(argsRaw interface{}) map[string]string {
	result := make(map[string]string)
	switch v := argsRaw.(type) {
	case map[string]interface{}:
		for k, val := range v {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				// Parse "KEY=VALUE" or "KEY"
				for i := 0; i < len(s); i++ {
					if s[i] == '=' {
						result[s[:i]] = s[i+1:]
						break
					}
				}
			}
		}
	}
	return result
}

func extractBuildLabels(labelsRaw interface{}) map[string]string {
	result := make(map[string]string)
	switch v := labelsRaw.(type) {
	case map[string]interface{}:
		for k, val := range v {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				for i := 0; i < len(s); i++ {
					if s[i] == '=' {
						result[s[:i]] = s[i+1:]
						break
					}
				}
			}
		}
	}
	return result
}

// ExtractServiceBuildSpecs parses raw compose YAML and extracts build metadata
// for all services that define a build block. It patches Dockerfiles and writes
// the patched copies to the cache directory.
func ExtractServiceBuildSpecs(composeData []byte, workingDir, cacheDir, projectName string) (map[string]*ServiceBuildSpec, error) {
	var cf ComposeFile
	if err := yaml.Unmarshal(composeData, &cf); err != nil {
		return nil, fmt.Errorf("failed to parse compose file for build specs: %w", err)
	}

	specs := make(map[string]*ServiceBuildSpec, len(cf.Services))
	for svcName, svc := range cf.Services {
		buildRaw, hasBuild := svc["build"]
		if !hasBuild {
			continue
		}

		spec, err := extractBuildSpec(svcName, buildRaw, workingDir, cacheDir, projectName)
		if err != nil {
			return nil, err
		}

		if img, ok := svc["image"].(string); ok && img != "" {
			spec.ImageTag = normalizeImage(img)
		}

		specs[svcName] = spec
	}
	return specs, nil
}
