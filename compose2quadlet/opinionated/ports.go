package opinionated

import (
	"fmt"
	"strconv"
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ApplyPortOffset(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if cfg.PortOffset == 0 {
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
				if d.Key != "PublishPort" {
					continue
				}
				for vi := range d.Values {
					original := d.Values[vi]
					d.Values[vi] = offsetPort(d.Values[vi], cfg.PortOffset)
					if cfg.Info != nil && d.Values[vi] != original {
						cfg.Info(fmt.Sprintf("Offset port in %s.container: %s → %s", units[ui].Name, original, d.Values[vi]))
					}
				}
			}
		}
	}

	return units
}

func offsetPort(port string, offset int) string {
	proto := ""
	if idx := strings.LastIndexByte(port, '/'); idx >= 0 {
		proto = port[idx:]
		port = port[:idx]
	}
	parts := strings.Split(port, ":")
	if len(parts) == 1 {
		port = parts[0]
	} else {
		hostIdx := len(parts) - 2
		if n, err := strconv.Atoi(parts[hostIdx]); err == nil && n <= 1024 {
			parts[hostIdx] = strconv.Itoa(n + offset)
		}
		port = strings.Join(parts, ":")
	}
	return port + proto
}
