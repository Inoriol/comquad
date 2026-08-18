package mapper

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
	"github.com/compose-spec/compose-go/v2/types"
)

func Builds(services types.Services, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	var units []c2qtypes.QuadletUnit
	for name, svc := range services {
		if svc.Build == nil {
			continue
		}
		if !cfg.PodmanVersion.AtLeast(5, 2) {
			continue
		}
		build := svc.Build
		var dirs []c2qtypes.Directive

		if build.Context != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "SetWorkingDirectory", Values: []string{build.Context}})
		}

		if cfg.NormalizeDockerfile && build.Dockerfile != "" && cfg.BuildCacheDir != "" {
			dockerfilePath := build.Dockerfile
			if !filepath.IsAbs(dockerfilePath) && build.Context != "" {
				dockerfilePath = filepath.Join(build.Context, dockerfilePath)
			}
			if !filepath.IsAbs(dockerfilePath) && cfg.WorkingDirectory != "" {
				dockerfilePath = filepath.Join(cfg.WorkingDirectory, dockerfilePath)
			}
			content, err := os.ReadFile(dockerfilePath)
			if err == nil {
				patched, err := PatchDockerfileFROM(bytes.NewReader(content))
				if err == nil {
					patchedPath := filepath.Join(cfg.BuildCacheDir, name+".Dockerfile")
					patchedOK := true
					if !cfg.DryRun {
						if err := os.MkdirAll(filepath.Dir(patchedPath), 0755); err != nil {
							patchedOK = false
							cfg.Warn(c2qtypes.Warning{
								Level:   c2qtypes.WarningDegraded,
								Service: name,
								Field:   "build.dockerfile",
								Message: fmt.Sprintf("failed to create Dockerfile cache directory: %v", err),
							})
						} else if err := os.WriteFile(patchedPath, patched, 0644); err != nil {
							patchedOK = false
							cfg.Warn(c2qtypes.Warning{
								Level:   c2qtypes.WarningDegraded,
								Service: name,
								Field:   "build.dockerfile",
								Message: fmt.Sprintf("failed to write patched Dockerfile: %v", err),
							})
						}
					}
					if patchedOK {
						if cfg.PatchedDockerfiles == nil {
							cfg.PatchedDockerfiles = make(map[string]string)
						}
						cfg.PatchedDockerfiles[name] = string(patched)
						build.Dockerfile = patchedPath
					}
				}
			}
		}

		if build.Dockerfile != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "File", Values: []string{build.Dockerfile}})
		} else if build.DockerfileInline != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "File", Values: []string{build.DockerfileInline}})
		}
		if build.Target != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "Target", Values: []string{build.Target}})
		}
		if build.Network != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "Network", Values: []string{build.Network}})
		}
		if build.NoCache {
			dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--no-cache"}})
		}
		for _, k := range sortedKeys(build.Labels) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Label", Values: []string{fmt.Sprintf("%s=%s", k, build.Labels[k])}})
		}
		for _, tag := range build.Tags {
			dirs = append(dirs, c2qtypes.Directive{Key: "ImageTag", Values: []string{tag}})
		}
		if len(build.Tags) == 0 {
			tag := name + ":latest"
			if cfg.ProjectName != "" {
				tag = cfg.ProjectName + "_" + name + ":latest"
			}
			dirs = append(dirs, c2qtypes.Directive{Key: "ImageTag", Values: []string{tag}})
		}
		for _, secret := range build.Secrets {
			if secret.Source != "" {
				dirs = append(dirs, c2qtypes.Directive{Key: "Secret", Values: []string{secret.Source}})
			}
		}
		if len(build.Args) > 0 {
			if cfg.PodmanVersion.AtLeast(5, 7) {
				for _, k := range sortedKeys(build.Args) {
					v := build.Args[k]
					if v != nil {
						dirs = append(dirs, c2qtypes.Directive{Key: "BuildArg", Values: []string{k + "=" + *v}})
					} else {
						dirs = append(dirs, c2qtypes.Directive{Key: "BuildArg", Values: []string{k}})
					}
				}
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "build.args",
					Message: "requires podman >= 5.7.0",
					Since:   "5.7.0",
				})
			}
		}

		if len(build.Platforms) > 0 {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: name,
				Field:   "build.platforms",
				Message: "multi-arch build not applicable",
			})
		}
		if len(build.ExtraHosts) > 0 {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: name,
				Field:   "build.extra_hosts",
				Message: "no podman build equivalent",
			})
		}

		units = append(units, c2qtypes.QuadletUnit{
			Type:     c2qtypes.UnitBuild,
			Name:     name,
			Sections: []c2qtypes.Section{{Name: c2qtypes.SectionBuild, Directives: dirs}},
		})
	}
	return units
}
