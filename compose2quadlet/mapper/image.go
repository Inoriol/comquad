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
		img := svc.Image
		if !cfg.SkipImageNormalization {
			img = normalizeImage(img)
		}
		dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{img}})

		useNative := cfg.PodmanVersion.AtLeast(4, 8)
		if !useNative && !cfg.PseudoQuadlet {
			continue
		}

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
			if useNative {
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
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "platform",
					Message: "platform not supported in pseudo-image mode",
				})
			}
		}

		if cfg.PodmanVersion.AtLeast(5, 5) && useNative {
			dirs = append(dirs, c2qtypes.Directive{Key: "Retry", Values: []string{strconv.Itoa(cfg.ImageRetry)}})
			dirs = append(dirs, c2qtypes.Directive{Key: "RetryDelay", Values: []string{fmt.Sprintf("%ds", cfg.ImageRetryDelay)}})
		}

		dirs = MergeDirectives(dirs, ExtractImageExtensions(svc, cfg))

		unitType := c2qtypes.UnitImage
		if !useNative {
			unitType = c2qtypes.UnitCQImage
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningDegraded,
				Service: name,
				Field:   "image",
				Message: "using pseudo-quadlet .cq-image fallback (requires podman >= 4.8.0 for native .image)",
				Since:   "4.8.0",
			})
		}

		units = append(units, c2qtypes.QuadletUnit{
			Type:     unitType,
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
