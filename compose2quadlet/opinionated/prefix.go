package opinionated

import c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"

func ApplyPrefix(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if cfg.FilePrefix == "" && cfg.ProjectName == "" {
		return units
	}
	var prefix string
	if cfg.ProjectName != "" {
		prefix = cfg.FilePrefix + cfg.ProjectName + "-"
	} else {
		prefix = cfg.FilePrefix
	}
	for i := range units {
		units[i].Name = prefix + units[i].Name
	}
	return units
}
