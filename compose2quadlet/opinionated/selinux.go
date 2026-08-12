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
						d.Values[vi] = d.Values[vi] + ",relabel=shared"
					} else {
						d.Values[vi] = d.Values[vi] + ",z"
					}
				}
			}
			}
		}
	}

	return units
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
