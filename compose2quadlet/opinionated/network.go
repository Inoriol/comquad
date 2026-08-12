package opinionated

import c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"

const defaultNetworkName = "default"

func ApplyDefaultNetwork(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if !cfg.DefaultNetwork {
		return units
	}

	hasNetwork := false
	for _, u := range units {
		if u.Type == c2qtypes.UnitNetwork {
			hasNetwork = true
			break
		}
		if u.Type == c2qtypes.UnitContainer {
			for _, s := range u.Sections {
				if s.Name == c2qtypes.SectionContainer {
					for _, d := range s.Directives {
						if d.Key == "Network" {
							hasNetwork = true
							break
						}
					}
				}
			}
		}
	}

	if hasNetwork {
		return units
	}

	netName := defaultNetworkName
	refName := defaultNetworkName + ".network"
	var pfx string
	if cfg.ProjectName != "" {
		pfx = cfg.FilePrefix + cfg.ProjectName + "-"
	} else {
		pfx = cfg.FilePrefix
	}
	if pfx != "" {
		netName = pfx + netName
		refName = pfx + refName
	}

	netUnit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitNetwork,
		Name: netName,
		Sections: []c2qtypes.Section{{
			Name: c2qtypes.SectionNetwork,
		}},
	}
	units = append(units, netUnit)

	for ui := range units {
		if units[ui].Type != c2qtypes.UnitContainer {
			continue
		}
		for si := range units[ui].Sections {
			if units[ui].Sections[si].Name != c2qtypes.SectionContainer {
				continue
			}
			found := false
			for _, d := range units[ui].Sections[si].Directives {
				if d.Key == "Network" {
					found = true
					break
				}
			}
			if !found {
				units[ui].Sections[si].Directives = append(units[ui].Sections[si].Directives,
					c2qtypes.Directive{Key: "Network", Values: []string{refName}})
			}
		}
	}

	return units
}
