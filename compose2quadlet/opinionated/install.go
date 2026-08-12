package opinionated

import c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"

func ApplyInstallSection(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if !cfg.InstallSection {
		return units
	}

	for ui := range units {
		hasInstall := false
		for _, s := range units[ui].Sections {
			if s.Name == c2qtypes.SectionInstall {
				hasInstall = true
				break
			}
		}
		if hasInstall {
			continue
		}
		units[ui].Sections = append(units[ui].Sections, c2qtypes.Section{
			Name: c2qtypes.SectionInstall,
			Directives: []c2qtypes.Directive{
				{Key: "WantedBy", Values: []string{"default.target"}},
			},
		})
	}

	return units
}
