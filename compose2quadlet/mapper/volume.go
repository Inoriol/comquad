package mapper

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func Volumes(volumes types.Volumes, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	var units []c2qtypes.QuadletUnit
	for name, vc := range volumes {
		if vc.External {
			continue
		}
		var dirs []c2qtypes.Directive
		if vc.Driver != "" && vc.Driver != "local" {
			if cfg.PodmanVersion.AtLeast(4, 7) {
				dirs = append(dirs, c2qtypes.Directive{Key: "Driver", Values: []string{vc.Driver}})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "volumes." + name + ".driver",
					Message: "requires podman >= 4.7.0",
					Since:   "4.7.0",
				})
			}
		}
		if len(vc.DriverOpts) > 0 {
			if cfg.PodmanVersion.AtLeast(6, 0) {
				for _, k := range sortedKeys(vc.DriverOpts) {
					dirs = append(dirs, c2qtypes.Directive{Key: "Options", Values: []string{k + "=" + vc.DriverOpts[k]}})
				}
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "volumes." + name + ".driver_opts",
					Message: "requires podman >= 6.0.0",
					Since:   "6.0.0",
				})
			}
		}
		if vc.Name != "" {
			if cfg.PodmanVersion.AtLeast(4, 7) {
				dirs = append(dirs, c2qtypes.Directive{Key: "VolumeName", Values: []string{vc.Name}})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "volumes." + name + ".name",
					Message: "requires podman >= 4.7.0",
					Since:   "4.7.0",
				})
			}
		}
		for _, k := range sortedKeys(vc.Labels) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Label", Values: []string{fmt.Sprintf("%s=%s", k, vc.Labels[k])}})
		}

		units = append(units, c2qtypes.QuadletUnit{
			Type:     c2qtypes.UnitVolume,
			Name:     name,
			Sections: []c2qtypes.Section{{Name: c2qtypes.SectionVolume, Directives: dirs}},
		})
	}
	return units
}
