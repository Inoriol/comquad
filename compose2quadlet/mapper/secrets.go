package mapper

import (
	"fmt"
	"os"
	"path/filepath"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
	"github.com/compose-spec/compose-go/v2/types"
)

func PremapSecrets(svc *types.ServiceConfig, secrets types.Secrets, configs types.Configs, cfg *c2qtypes.Config) ([]c2qtypes.Directive, []c2qtypes.QuadletUnit) {
	var containerDirs []c2qtypes.Directive
	var secretUnits []c2qtypes.QuadletUnit

	for _, ref := range svc.Secrets {
		def, ok := secrets[ref.Source]
		if !ok {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "secrets." + ref.Source,
				Message: fmt.Sprintf("secret %s not defined in top-level secrets", ref.Source),
			})
			continue
		}
		dirs, units := processSecret(svc.Name, ref, def, cfg)
		containerDirs = append(containerDirs, dirs...)
		secretUnits = append(secretUnits, units...)
	}

	for _, ref := range svc.Configs {
		def, ok := configs[ref.Source]
		if !ok {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "configs." + ref.Source,
				Message: fmt.Sprintf("config %s not defined in top-level configs", ref.Source),
			})
			continue
		}
		d := processConfig(svc.Name, ref, def)
		containerDirs = append(containerDirs, d...)
	}

	svc.Secrets = nil
	svc.Configs = nil

	return containerDirs, secretUnits
}

func processSecret(serviceName string, ref types.ServiceSecretConfig, def types.SecretConfig, cfg *c2qtypes.Config) ([]c2qtypes.Directive, []c2qtypes.QuadletUnit) {
	target := ref.Target
	if target == "" {
		target = "/run/secrets/" + ref.Source
	}

	if bool(def.External) {
		if !cfg.PodmanVersion.AtLeast(4, 5) {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: serviceName,
				Field:   "secrets." + ref.Source,
				Message: "external secrets require podman >= 4.5.0",
				Since:   "4.5.0",
			})
			return nil, nil
		}
		name := def.Name
		if name == "" {
			name = ref.Source
		}
		return []c2qtypes.Directive{{Key: "Secret", Values: []string{name}}}, nil
	}

	hasCustomPerms := ref.UID != "" || ref.GID != "" || ref.Mode != nil

	if hasCustomPerms {
		return processSecretVolume(serviceName, ref, def, cfg), nil
	}

	return processSecretTmpfs(serviceName, ref, def, cfg)
}

func processSecretTmpfs(serviceName string, ref types.ServiceSecretConfig, def types.SecretConfig, cfg *c2qtypes.Config) ([]c2qtypes.Directive, []c2qtypes.QuadletUnit) {
	if def.File != "" {
		secretUnit := c2qtypes.QuadletUnit{
			Type: c2qtypes.UnitCQSecret,
			Name: ref.Source,
			Sections: []c2qtypes.Section{
				{
					Name: c2qtypes.SectionSecret,
					Directives: []c2qtypes.Directive{
						{Key: "Source", Values: []string{"file"}},
						{Key: "Path", Values: []string{def.File}},
					},
				},
			},
		}
		containerDir := c2qtypes.Directive{
			Key:    "Secret",
			Values: []string{ref.Source + ".secretref"},
		}
		return []c2qtypes.Directive{containerDir}, []c2qtypes.QuadletUnit{secretUnit}
	}

	if def.Environment != "" {
		secretUnit := c2qtypes.QuadletUnit{
			Type: c2qtypes.UnitCQSecret,
			Name: ref.Source,
			Sections: []c2qtypes.Section{
				{
					Name: c2qtypes.SectionSecret,
					Directives: []c2qtypes.Directive{
						{Key: "Source", Values: []string{"environment"}},
						{Key: "Environment", Values: []string{def.Environment}},
					},
				},
			},
		}
		containerDir := c2qtypes.Directive{
			Key:    "Secret",
			Values: []string{ref.Source + ".secretref"},
		}
		return []c2qtypes.Directive{containerDir}, []c2qtypes.QuadletUnit{secretUnit}
	}

	return nil, nil
}

func processSecretVolume(serviceName string, ref types.ServiceSecretConfig, def types.SecretConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	target := ref.Target
	if target == "" {
		target = "/run/secrets/" + ref.Source
	}

	if def.File != "" {
		return []c2qtypes.Directive{{Key: "Volume", Values: []string{def.File + ":" + target + ":ro"}}}
	}

	if def.Environment != "" {
		if cfg.SecretsDir == "" {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningDegraded,
				Service: serviceName,
				Field:   "secrets." + ref.Source,
				Message: "environment secrets with custom permissions require WithSecretsDirectory()",
			})
			return nil
		}
		val := os.Getenv(def.Environment)
		if val == "" {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningDegraded,
				Service: serviceName,
				Field:   "secrets." + ref.Source,
				Message: fmt.Sprintf("environment variable %q is not set or empty", def.Environment),
			})
			return nil
		}
		sourcePath := filepath.Join(cfg.SecretsDir, ref.Source)
		if !cfg.DryRun {
			if err := os.MkdirAll(cfg.SecretsDir, 0700); err != nil {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningDegraded,
					Service: serviceName,
					Field:   "secrets." + ref.Source,
					Message: fmt.Sprintf("failed to create secrets directory: %v", err),
				})
				return nil
			}
			if err := os.WriteFile(sourcePath, []byte(val), 0600); err != nil {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningDegraded,
					Service: serviceName,
					Field:   "secrets." + ref.Source,
					Message: fmt.Sprintf("failed to write secret file: %v", err),
				})
				return nil
			}
		}
		return []c2qtypes.Directive{{Key: "Volume", Values: []string{sourcePath + ":" + target + ":ro"}}}
	}

	return nil
}

func processConfig(serviceName string, ref types.ServiceConfigObjConfig, def types.ConfigObjConfig) []c2qtypes.Directive {
	target := ref.Target
	if target == "" {
		target = "/" + ref.Source
	}

	source := def.File
	if source == "" {
		source = def.Name
	}
	if source == "" {
		source = ref.Source
	}

	mount := fmt.Sprintf("type=bind,source=%s,destination=%s", source, target)
	if ref.UID != "" {
		mount += ",uid=" + ref.UID
	}
	if ref.GID != "" {
		mount += ",gid=" + ref.GID
	}
	if ref.Mode != nil {
		mount += fmt.Sprintf(",mode=%04o", os.FileMode(*ref.Mode))
	}

	return []c2qtypes.Directive{{Key: "Mount", Values: []string{mount}}}
}
