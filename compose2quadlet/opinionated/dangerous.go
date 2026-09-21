package opinionated

import (
	"os"
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

var defaultDangerousPaths = []string{
	"/etc",
	"/var",
	"/home",
	"/usr",
	"/sys",
	"/proc",
	"/boot",
	"/root",
}

func WarnDangerousBindMounts(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	if !cfg.SelinuxContext {
		return units
	}

	dangerousPaths := defaultDangerousPaths
	if envPaths := os.Getenv("COMQUAD_SELINUX_DANGEROUS_PATHS"); envPaths != "" {
		dangerousPaths = strings.Split(envPaths, ",")
		for i := range dangerousPaths {
			dangerousPaths[i] = strings.TrimSpace(dangerousPaths[i])
		}
	}

	for ui := range units {
		if units[ui].Type != c2qtypes.UnitContainer {
			continue
		}
		serviceName := extractServiceName(units[ui].Name, cfg)
		for si := range units[ui].Sections {
			if units[ui].Sections[si].Name != c2qtypes.SectionContainer {
				continue
			}
			for di := range units[ui].Sections[si].Directives {
				d := &units[ui].Sections[si].Directives[di]
				if d.Key != "Mount" && d.Key != "Volume" {
					continue
				}
				for _, v := range d.Values {
					sourcePath := extractSourcePath(v, d.Key)
					if sourcePath == "" {
						continue
					}
					for _, dp := range dangerousPaths {
						if dp == "" {
							continue
						}
						if sourcePath == dp || strings.HasPrefix(sourcePath, dp+"/") {
							cfg.Warn(c2qtypes.Warning{
								Level:   c2qtypes.WarningDegraded,
								Service: serviceName,
								Field:   "volumes",
								Message: "SELinux relabeling on dangerous host path " + sourcePath + " may cause system issues",
							})
							break
						}
					}
				}
			}
		}
	}

	return units
}

func extractSourcePath(value, key string) string {
	if key == "Mount" {
		for _, part := range strings.Split(value, ",") {
			if strings.HasPrefix(part, "source=") {
				return strings.TrimPrefix(part, "source=")
			}
		}
		return ""
	}
	colon := strings.IndexByte(value, ':')
	if colon > 0 {
		return value[:colon]
	}
	return ""
}

func extractServiceName(unitName string, cfg *c2qtypes.Config) string {
	prefix := cfg.FilePrefix
	if cfg.ProjectName != "" {
		prefix += cfg.ProjectName + "-"
	}
	if prefix != "" && strings.HasPrefix(unitName, prefix) {
		return strings.TrimPrefix(unitName, prefix)
	}
	return unitName
}
