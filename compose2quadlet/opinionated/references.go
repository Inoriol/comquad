package opinionated

import (
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ApplyReferences(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if cfg.FilePrefix == "" && cfg.ProjectName == "" {
		return units
	}
	var prefix string
	if cfg.ProjectName != "" {
		prefix = cfg.FilePrefix + cfg.ProjectName + "-"
	} else {
		prefix = cfg.FilePrefix
	}
	refKeys := map[string][]string{
		"Network": {".network"},
		"Volume":  {".volume"},
		"Image":   {".image", ".build", ".cq-image", ".cq-build"},
		"Mount":   {".volume", ".image", ".cq-image"},
		"Secret":  {".secretref"},
	}
	unitRefKeys := map[string]string{
		"After":    ".container",
		"Requires": ".container",
		"Wants":    ".container",
		"BindsTo":  ".container",
		"PartOf":   ".container",
	}
	pseudoRefMap := map[string]string{
		".cq-image": "-image.service",
		".cq-build": "-build.service",
	}

	for ui := range units {
		var secretServices []string

		for si := range units[ui].Sections {
			for di := range units[ui].Sections[si].Directives {
				d := &units[ui].Sections[si].Directives[di]

				if suffixes, ok := refKeys[d.Key]; ok {
					for vi := range d.Values {
						head, tail := splitRefHead(d.Values[vi])
						if d.Key == "Network" && strings.HasSuffix(head, ".network") {
							logical := strings.TrimSuffix(head, ".network")
							if actual, ok := cfg.ExternalNetworks[logical]; ok {
								d.Values[vi] = actual + tail
								continue
							}
						}
						if d.Key == "Volume" && strings.HasSuffix(head, ".volume") {
							logical := strings.TrimSuffix(head, ".volume")
							if actual, ok := cfg.ExternalVolumes[logical]; ok {
								d.Values[vi] = actual + tail
								continue
							}
						}
					if d.Key == "Secret" && strings.HasSuffix(head, ".secretref") {
						secretName := strings.TrimSuffix(head, ".secretref")
						prefixedSecretName := prefix + secretName
						d.Values[vi] = prefixedSecretName
						secretServices = append(secretServices, prefixedSecretName+"-secret.service")
						continue
					}
						for _, suffix := range suffixes {
							if strings.HasSuffix(head, suffix) {
								d.Values[vi] = prefix + head + tail
								break
							}
						}
					}
				}

			if suffix, ok := unitRefKeys[d.Key]; ok {
				for vi := range d.Values {
					if !strings.Contains(d.Values[vi], ".") {
						d.Values[vi] = d.Values[vi] + suffix
					}
					for cqExt, svcSuffix := range pseudoRefMap {
						if strings.HasSuffix(d.Values[vi], cqExt) {
							name := strings.TrimSuffix(d.Values[vi], cqExt)
							d.Values[vi] = name + svcSuffix
							break
						}
					}
					if !strings.HasPrefix(d.Values[vi], prefix) {
						d.Values[vi] = prefix + d.Values[vi]
					}
				}
			}
			}
		}

		if len(secretServices) > 0 && units[ui].Type == c2qtypes.UnitContainer {
			addSecretDependencies(units, ui, secretServices)
		}
	}
	return units
}

func addSecretDependencies(units []c2qtypes.QuadletUnit, unitIdx int, secretServices []string) {
	var unitSection *c2qtypes.Section
	for i := range units[unitIdx].Sections {
		if units[unitIdx].Sections[i].Name == c2qtypes.SectionUnit {
			unitSection = &units[unitIdx].Sections[i]
			break
		}
	}

	if unitSection == nil {
		units[unitIdx].Sections = append([]c2qtypes.Section{{Name: c2qtypes.SectionUnit}}, units[unitIdx].Sections...)
		unitSection = &units[unitIdx].Sections[0]
	}

	for _, svc := range secretServices {
		unitSection.Directives = append(unitSection.Directives, c2qtypes.Directive{Key: "After", Values: []string{svc}})
		unitSection.Directives = append(unitSection.Directives, c2qtypes.Directive{Key: "Requires", Values: []string{svc}})
	}
}

func splitRefHead(val string) (head, tail string) {
	colon := strings.IndexByte(val, ':')
	if colon < 0 {
		return val, ""
	}
	return val[:colon], val[colon:]
}
