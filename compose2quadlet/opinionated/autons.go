package opinionated

import (
	"fmt"
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

const autoNSRangeSize = 65535

func ApplyAutoNS(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if cfg.AutoNS == nil {
		return units
	}

	prefix := cfg.FilePrefix + cfg.ProjectName + "-"
	anchorUnitName := prefix + cfg.AutoNS.AnchorService
	anchorServiceRef := anchorUnitName + ".service"
	anchorContainerName := resolveContainerName(units, anchorUnitName, cfg.AutoNS.AnchorService, cfg)

	autoNSAnchorArg := fmt.Sprintf("--userns=auto:uidmapping=0:%d:%d", cfg.AutoNS.StackBaseUID, autoNSRangeSize)
	autoNSJoinArg := fmt.Sprintf("--userns=container:%s", anchorContainerName)
	idmapOpt := "idmap"

	for ui := range units {
		if units[ui].Type != c2qtypes.UnitContainer {
			continue
		}

		isAnchor := units[ui].Name == anchorUnitName

		for si := range units[ui].Sections {
			sec := &units[ui].Sections[si]

			if sec.Name == c2qtypes.SectionContainer {
				for di := range sec.Directives {
					d := &sec.Directives[di]
					if d.Key == "Volume" {
						for vi := range d.Values {
							d.Values[vi] = appendVolumeOption(d.Values[vi], idmapOpt)
						}
					}
					if d.Key == "Mount" {
						for vi := range d.Values {
							d.Values[vi] = appendMountOption(d.Values[vi], idmapOpt)
						}
					}
				}

				if isAnchor {
					sec.Directives = append(sec.Directives,
						c2qtypes.Directive{Key: "PodmanArgs", Values: []string{autoNSAnchorArg}})
				} else {
					sec.Directives = append(sec.Directives,
						c2qtypes.Directive{Key: "PodmanArgs", Values: []string{autoNSJoinArg}})
				}
			}

			if sec.Name == c2qtypes.SectionUnit && !isAnchor {
				sec.Directives = append(sec.Directives,
					c2qtypes.Directive{Key: "After", Values: []string{anchorServiceRef}})
				sec.Directives = append(sec.Directives,
					c2qtypes.Directive{Key: "Requires", Values: []string{anchorServiceRef}})
			}
		}

		if !isAnchor {
			hasUnitSection := false
			for si := range units[ui].Sections {
				if units[ui].Sections[si].Name == c2qtypes.SectionUnit {
					hasUnitSection = true
					break
				}
			}
			if !hasUnitSection {
				units[ui].Sections = append([]c2qtypes.Section{{
					Name: c2qtypes.SectionUnit,
					Directives: []c2qtypes.Directive{
						{Key: "After", Values: []string{anchorServiceRef}},
						{Key: "Requires", Values: []string{anchorServiceRef}},
					},
				}}, units[ui].Sections...)
			}
		}
	}

	return units
}

func resolveContainerName(units []c2qtypes.QuadletUnit, unitName string, composeServiceName string, cfg *c2qtypes.Config) string {
	for ui := range units {
		if units[ui].Name != unitName || units[ui].Type != c2qtypes.UnitContainer {
			continue
		}
		for si := range units[ui].Sections {
			if units[ui].Sections[si].Name != c2qtypes.SectionContainer {
				continue
			}
			for di := range units[ui].Sections[si].Directives {
				if units[ui].Sections[si].Directives[di].Key == "ContainerName" && len(units[ui].Sections[si].Directives[di].Values) > 0 {
					return units[ui].Sections[si].Directives[di].Values[0]
				}
			}
		}
	}
	return cfg.ProjectName + "-" + composeServiceName
}

func appendVolumeOption(value string, opt string) string {
	if strings.Contains(value, "idmap") {
		return value
	}
	return value + "," + opt
}

func appendMountOption(value string, opt string) string {
	if strings.Contains(value, "idmap") {
		return value
	}
	if idx := strings.Index(value, "options="); idx != -1 {
		existing := value[idx+8:]
		if commaIdx := strings.Index(existing, ","); commaIdx != -1 {
			existing = existing[:commaIdx]
		}
		return value + "," + opt
	}
	return value + ",options=" + opt
}
