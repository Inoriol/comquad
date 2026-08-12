package mapper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func Images(services types.Services, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	var units []c2qtypes.QuadletUnit
	for name, svc := range services {
		if svc.Image == "" {
			continue
		}
		var dirs []c2qtypes.Directive
		dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{normalizeImage(svc.Image)}})

		if svc.PullPolicy != "" {
			policy := composePolicyToQuadlet(svc.PullPolicy)
			if policy != "" {
				if cfg.PodmanVersion.AtLeast(5, 6) {
					dirs = append(dirs, c2qtypes.Directive{Key: "Policy", Values: []string{policy}})
				} else {
					cfg.Warn(c2qtypes.Warning{
						Level:   c2qtypes.WarningSkipped,
						Service: name,
						Field:   "pull_policy",
						Message: "Policy= in .image requires podman >= 5.6.0",
						Since:   "5.6.0",
					})
				}
			}
		}

		if svc.Platform != "" {
			parts := strings.SplitN(svc.Platform, "/", 3)
			if len(parts) >= 1 && parts[0] != "" {
				dirs = append(dirs, c2qtypes.Directive{Key: "OS", Values: []string{parts[0]}})
			}
			if len(parts) >= 2 && parts[1] != "" {
				dirs = append(dirs, c2qtypes.Directive{Key: "Arch", Values: []string{parts[1]}})
			}
			if len(parts) >= 3 && parts[2] != "" {
				dirs = append(dirs, c2qtypes.Directive{Key: "Variant", Values: []string{parts[2]}})
			}
		}

		if cfg.PodmanVersion.AtLeast(5, 5) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Retry", Values: []string{strconv.Itoa(cfg.ImageRetry)}})
			dirs = append(dirs, c2qtypes.Directive{Key: "RetryDelay", Values: []string{fmt.Sprintf("%ds", cfg.ImageRetryDelay)}})
		}

		units = append(units, c2qtypes.QuadletUnit{
			Type:     c2qtypes.UnitImage,
			Name:     name,
			Sections: []c2qtypes.Section{{Name: c2qtypes.SectionImage, Directives: dirs}},
		})
	}
	return units
}

func composePolicyToQuadlet(policy string) string {
	switch policy {
	case "always", "missing", "never":
		return policy
	case "if_not_present":
		return "missing"
	default:
		return ""
	}
}
