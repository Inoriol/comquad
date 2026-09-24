package opinionated

import (
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ApplySELinux(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if !cfg.SelinuxContext {
		return units
	}

	for ui := range units {
		if units[ui].Type != c2qtypes.UnitContainer {
			continue
		}
		for si := range units[ui].Sections {
			if units[ui].Sections[si].Name != c2qtypes.SectionContainer {
				continue
			}
			for di := range units[ui].Sections[si].Directives {
				d := &units[ui].Sections[si].Directives[di]

			if d.Key == "Volume" || d.Key == "Mount" {
				for vi := range d.Values {
					if hasSELinuxContext(d.Values[vi]) {
						continue
					}
					if d.Key == "Mount" {
						if isSharedMount(d.Values[vi], cfg.SharedVolumes) {
							d.Values[vi] = d.Values[vi] + ",relabel=shared"
						} else {
							d.Values[vi] = d.Values[vi] + ",relabel=private"
						}
					} else {
						if isSharedVolume(d.Values[vi], cfg.SharedVolumes) {
							d.Values[vi] = d.Values[vi] + ",z"
						} else {
							d.Values[vi] = d.Values[vi] + ",Z"
						}
					}
				}
			}
			}
		}
	}

	return units
}

func isSharedMount(value string, shared map[string]bool) bool {
	sourceIdx := strings.Index(value, "source=")
	if sourceIdx == -1 {
		return false
	}
	rest := value[sourceIdx+7:]
	commaIdx := strings.Index(rest, ",")
	var source string
	if commaIdx == -1 {
		source = rest
	} else {
		source = rest[:commaIdx]
	}
	return shared[source]
}

func isSharedVolume(value string, shared map[string]bool) bool {
	colonIdx := strings.Index(value, ":")
	if colonIdx == -1 {
		return false
	}
	volName := value[:colonIdx]
	if strings.HasSuffix(volName, ".volume") {
		volName = strings.TrimSuffix(volName, ".volume")
		if strings.HasPrefix(volName, "cq-") {
			parts := strings.SplitN(volName[3:], "-", 2)
			if len(parts) == 2 {
				volName = parts[1]
			}
		} else if strings.Contains(volName, "-") {
			parts := strings.SplitN(volName, "-", 2)
			if len(parts) == 2 {
				if shared[parts[1]] {
					return true
				}
			}
		}
	}
	return shared[volName]
}

func hasSELinuxContext(v string) bool {
	if strings.Contains(v, "relabel=") {
		return true
	}
	lastColon := strings.LastIndex(v, ":")
	if lastColon == -1 {
		return false
	}
	rest := v[lastColon+1:]
	return rest == "z" || rest == "Z" || strings.HasPrefix(rest, "z,") || strings.HasPrefix(rest, "Z,")
}
