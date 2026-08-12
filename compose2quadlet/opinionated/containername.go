package opinionated

import (
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ApplyContainerName(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	prefix := cfg.FilePrefix + cfg.ProjectName + "-"

	for ui := range units {
		if units[ui].Type != c2qtypes.UnitContainer {
			continue
		}

		svcName := strings.TrimPrefix(units[ui].Name, prefix)
		containerName := cfg.ProjectName + "-" + svcName

		for si := range units[ui].Sections {
			if units[ui].Sections[si].Name != c2qtypes.SectionContainer {
				continue
			}

			found := false
			for _, d := range units[ui].Sections[si].Directives {
				if d.Key == "ContainerName" {
					found = true
					break
				}
			}
			if !found {
				units[ui].Sections[si].Directives = append(units[ui].Sections[si].Directives,
					c2qtypes.Directive{Key: "ContainerName", Values: []string{containerName}})
			}
			break
		}
	}

	return units
}
