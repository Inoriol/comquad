package opinionated

import (
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ApplyAutoUpdate(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if !cfg.AutoUpdate {
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
			hasBuild := false
			for _, d := range units[ui].Sections[si].Directives {
				if d.Key == "Image" && len(d.Values) > 0 && strings.HasSuffix(d.Values[0], ".build") {
					hasBuild = true
				}
			}
			if hasBuild {
				continue
			}
			found := false
			for _, d := range units[ui].Sections[si].Directives {
				if d.Key == "AutoUpdate" {
					found = true
					break
				}
			}
			if !found {
				units[ui].Sections[si].Directives = append(units[ui].Sections[si].Directives,
					c2qtypes.Directive{Key: "AutoUpdate", Values: []string{"registry"}})
			}
		}
	}

	return units
}
