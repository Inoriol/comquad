package mapper

import (
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func SecurityOpts(opts []string, serviceName string, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	for _, opt := range opts {
		switch {
		case opt == "no-new-privileges":
			dirs = append(dirs, c2qtypes.Directive{Key: "NoNewPrivileges"})

		case opt == "label=disable":
			dirs = append(dirs, c2qtypes.Directive{Key: "SecurityLabelDisable"})

		case opt == "label=nested":
			if cfg.PodmanVersion.AtLeast(4, 6) {
				dirs = append(dirs, c2qtypes.Directive{Key: "SecurityLabelNested"})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: serviceName,
					Field:   "security_opt",
					Message: "label=nested requires podman >= 4.6.0",
					Since:   "4.6.0",
				})
			}

		case strings.HasPrefix(opt, "seccomp="):
			dirs = append(dirs, c2qtypes.Directive{Key: "SeccompProfile", Values: []string{opt[len("seccomp="):]}})

		case strings.HasPrefix(opt, "label=type:"):
			dirs = append(dirs, c2qtypes.Directive{Key: "SecurityLabelType", Values: []string{opt[len("label=type:"):]}})

		case strings.HasPrefix(opt, "label=level:"):
			dirs = append(dirs, c2qtypes.Directive{Key: "SecurityLabelLevel", Values: []string{opt[len("label=level:"):]}})

		case strings.HasPrefix(opt, "label=filetype:"):
			dirs = append(dirs, c2qtypes.Directive{Key: "SecurityLabelFileType", Values: []string{opt[len("label=filetype:"):]}})

		case strings.HasPrefix(opt, "mask="):
			if cfg.PodmanVersion.AtLeast(4, 6) {
				dirs = append(dirs, c2qtypes.Directive{Key: "Mask", Values: []string{opt[len("mask="):]}})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: serviceName,
					Field:   "security_opt",
					Message: "mask requires podman >= 4.6.0",
					Since:   "4.6.0",
				})
			}

		case strings.HasPrefix(opt, "unmask="):
			if cfg.PodmanVersion.AtLeast(4, 6) {
				dirs = append(dirs, c2qtypes.Directive{Key: "Unmask", Values: []string{opt[len("unmask="):]}})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: serviceName,
					Field:   "security_opt",
					Message: "unmask requires podman >= 4.6.0",
					Since:   "4.6.0",
				})
			}

		case strings.HasPrefix(opt, "apparmor="):
			if cfg.PodmanVersion.AtLeast(5, 8) {
				dirs = append(dirs, c2qtypes.Directive{Key: "AppArmor", Values: []string{opt[len("apparmor="):]}})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: serviceName,
					Field:   "security_opt",
					Message: "apparmor requires podman >= 5.8.0",
					Since:   "5.8.0",
				})
			}
		}
	}

	return dirs
}
