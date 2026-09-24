package opinionated

import (
	"os"
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ApplySpecifiers(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if !cfg.SystemdSpecifiers {
		return units
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
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
						d.Values[vi] = shortenPath(d.Values[vi], homeDir, d.Key)
					}
				}
			}
		}
	}

	return units
}

func shortenPath(value, homeDir, directiveKey string) string {
	if directiveKey == "Mount" {
		return shortenMountPath(value, homeDir)
	}
	return shortenVolumePath(value, homeDir)
}

func shortenMountPath(value, homeDir string) string {
	sourceIdx := strings.Index(value, "source=")
	if sourceIdx == -1 {
		return value
	}

	rest := value[sourceIdx+7:]
	commaIdx := strings.Index(rest, ",")
	var source, remainder string
	if commaIdx == -1 {
		source = rest
		remainder = ""
	} else {
		source = rest[:commaIdx]
		remainder = rest[commaIdx:]
	}

	if strings.HasPrefix(source, homeDir+"/") {
		source = "%h" + source[len(homeDir):]
	} else if source == homeDir {
		source = "%h"
	}

	return value[:sourceIdx+7] + source + remainder
}

func shortenVolumePath(value, homeDir string) string {
	colonIdx := strings.Index(value, ":")
	if colonIdx == -1 {
		return value
	}

	hostPath := value[:colonIdx]
	rest := value[colonIdx:]

	if strings.HasPrefix(hostPath, homeDir+"/") {
		hostPath = "%h" + hostPath[len(homeDir):]
	} else if hostPath == homeDir {
		hostPath = "%h"
	}

	return hostPath + rest
}
