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
						if d.Key == "Volume" && strings.Contains(head, ".secretref") {
							secretName := strings.TrimSuffix(head, ".secretref")
							secretName = strings.TrimPrefix(secretName, "/run/credentials/")
							d.Values[vi] = "/run/credentials/" + prefix + secretName + "-secret.service/" + secretName + tail
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
	}
	return units
}

func splitRefHead(val string) (head, tail string) {
	colon := strings.IndexByte(val, ':')
	if colon < 0 {
		return val, ""
	}
	return val[:colon], val[colon:]
}
