package mapper

import (
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ExtractContainerExtensions(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	return extractExtensions(svc.Extensions, "x-container", c2qtypes.SectionContainer, svc.Name, cfg)
}

func ExtractImageExtensions(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	return extractExtensions(svc.Extensions, "x-image", c2qtypes.SectionImage, svc.Name, cfg)
}

func ExtractBuildExtensions(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	return extractExtensions(svc.Extensions, "x-build", c2qtypes.SectionBuild, svc.Name, cfg)
}

func ExtractSystemdExtensions(svc types.ServiceConfig, cfg *c2qtypes.Config) (service []c2qtypes.Directive, unit []c2qtypes.Directive) {
	ext, ok := svc.Extensions["x-systemd"]
	if !ok {
		return nil, nil
	}

	extMap, ok := ext.(map[string]interface{})
	if !ok {
		cfg.Warn(c2qtypes.Warning{
			Level:   c2qtypes.WarningSkipped,
			Service: svc.Name,
			Field:   "x-systemd",
			Message: "must be a map with Service and/or Unit keys",
		})
		return nil, nil
	}

	if svcSection, ok := extMap["Service"]; ok {
		service = extractDirectivesFromMap(svcSection, c2qtypes.SectionService, svc.Name, cfg)
	}

	if unitSection, ok := extMap["Unit"]; ok {
		unit = extractDirectivesFromMap(unitSection, c2qtypes.SectionUnit, svc.Name, cfg)
	}

	return service, unit
}

func ExtractNetworkExtensions(name string, nc types.NetworkConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	return extractExtensions(nc.Extensions, "x-network", c2qtypes.SectionNetwork, name, cfg)
}

func ExtractVolumeExtensions(name string, vc types.VolumeConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	return extractExtensions(vc.Extensions, "x-volume", c2qtypes.SectionVolume, name, cfg)
}

func extractExtensions(extensions types.Extensions, key, section, serviceName string, cfg *c2qtypes.Config) []c2qtypes.Directive {
	ext, ok := extensions[key]
	if !ok {
		return nil
	}

	return extractDirectivesFromMap(ext, section, serviceName, cfg)
}

func extractDirectivesFromMap(ext interface{}, section, serviceName string, cfg *c2qtypes.Config) []c2qtypes.Directive {
	extMap, ok := ext.(map[string]interface{})
	if !ok {
		cfg.Warn(c2qtypes.Warning{
			Level:   c2qtypes.WarningSkipped,
			Service: serviceName,
			Field:   section,
			Message: fmt.Sprintf("x-extension must be a map for section [%s]", section),
		})
		return nil
	}

	var dirs []c2qtypes.Directive
	for _, key := range sortedKeys(extMap) {
		value := extMap[key]
		directiveKey := strings.TrimSuffix(key, "=")

		if !isValidDirective(section, directiveKey+"=") {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: serviceName,
				Field:   directiveKey,
				Message: fmt.Sprintf("unknown directive for [%s], may require newer podman", section),
			})
		}

		values := formatExtensionValue(value)
		dirs = append(dirs, c2qtypes.Directive{
			Key:    directiveKey,
			Values: values,
		})
	}

	return dirs
}

func formatExtensionValue(value interface{}) []string {
	switch v := value.(type) {
	case string:
		return []string{v}
	case bool:
		if v {
			return []string{"true"}
		}
		return []string{"false"}
	case int, int32, int64:
		return []string{fmt.Sprintf("%d", v)}
	case float32, float64:
		return []string{fmt.Sprintf("%v", v)}
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else {
				result = append(result, fmt.Sprintf("%v", item))
			}
		}
		return result
	default:
		return []string{fmt.Sprintf("%v", v)}
	}
}

func MergeDirectives(base, override []c2qtypes.Directive) []c2qtypes.Directive {
	if len(override) == 0 {
		return base
	}
	if len(base) == 0 {
		return override
	}

	directiveMap := make(map[string]*c2qtypes.Directive)
	var order []string

	for i := range base {
		key := base[i].Key
		if _, exists := directiveMap[key]; !exists {
			order = append(order, key)
		}
		directiveMap[key] = &base[i]
	}

	for i := range override {
		key := override[i].Key
		if _, exists := directiveMap[key]; !exists {
			order = append(order, key)
		}
		directiveMap[key] = &override[i]
	}

	var result []c2qtypes.Directive
	for _, key := range order {
		result = append(result, *directiveMap[key])
	}

	return result
}
