package opinionated

import (
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ApplyNetworkAliases(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if !cfg.NetworkAliases {
		return units
	}

	prefix := cfg.FilePrefix + cfg.ProjectName + "-"

	for ui := range units {
		if units[ui].Type != c2qtypes.UnitContainer {
			continue
		}

		svcName := strings.TrimPrefix(units[ui].Name, prefix)

		for si := range units[ui].Sections {
			if units[ui].Sections[si].Name != c2qtypes.SectionContainer {
				continue
			}

			sec := &units[ui].Sections[si]

			seen := map[string]bool{}
			addAlias := func(name string) {
				if !seen[name] {
					seen[name] = true
					sec.Directives = append(sec.Directives,
						c2qtypes.Directive{Key: "NetworkAlias", Values: []string{name}})
				}
			}

			addAlias(svcName)
			if cfg.ProjectName != "" {
				addAlias(cfg.ProjectName + "-" + svcName)
			}
		}
	}

	return units
}
