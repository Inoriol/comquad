package opinionated

import (
	"sort"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ApplyLabels(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if len(cfg.Labels) == 0 {
		return units
	}

	keys := make([]string, 0, len(cfg.Labels))
	for k := range cfg.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for ui := range units {
		for si := range units[ui].Sections {
			sec := &units[ui].Sections[si]
			if sec.Name == c2qtypes.SectionService || sec.Name == c2qtypes.SectionUnit ||
				sec.Name == c2qtypes.SectionImage {
				continue
			}
			for _, k := range keys {
				label := k + "=" + cfg.Labels[k]
				found := false
				for _, d := range sec.Directives {
					if d.Key == "Label" {
						for _, dv := range d.Values {
							if dv == label {
								found = true
								break
							}
						}
					}
				}
				if !found {
					sec.Directives = append(sec.Directives,
						c2qtypes.Directive{Key: "Label", Values: []string{label}})
				}
			}
		}
	}

	return units
}
