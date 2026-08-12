package mapper

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func Networks(networks types.Networks, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	var units []c2qtypes.QuadletUnit
	for name, nc := range networks {
		if nc.External {
			continue
		}
		var dirs []c2qtypes.Directive
		if nc.Driver != "" && nc.Driver != "bridge" {
			dirs = append(dirs, c2qtypes.Directive{Key: "Driver", Values: []string{nc.Driver}})
		}
		if len(nc.DriverOpts) > 0 {
			if cfg.PodmanVersion.AtLeast(6, 0) {
				for _, k := range sortedKeys(nc.DriverOpts) {
					dirs = append(dirs, c2qtypes.Directive{Key: "Options", Values: []string{k + "=" + nc.DriverOpts[k]}})
				}
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "networks." + name + ".driver_opts",
					Message: "requires podman >= 6.0.0",
					Since:   "6.0.0",
				})
			}
		}
		if nc.Ipam.Driver != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "IPAMDriver", Values: []string{nc.Ipam.Driver}})
		}
		for _, pool := range nc.Ipam.Config {
			if pool.Subnet != "" {
				dirs = append(dirs, c2qtypes.Directive{Key: "Subnet", Values: []string{pool.Subnet}})
			}
			if pool.Gateway != "" {
				dirs = append(dirs, c2qtypes.Directive{Key: "Gateway", Values: []string{pool.Gateway}})
			}
			if pool.IPRange != "" {
				dirs = append(dirs, c2qtypes.Directive{Key: "IPRange", Values: []string{pool.IPRange}})
			}
		}
		if nc.Internal {
			dirs = append(dirs, c2qtypes.Directive{Key: "Internal", Values: []string{"true"}})
		}
		if nc.Name != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "NetworkName", Values: []string{nc.Name}})
		}
		if nc.EnableIPv6 != nil && *nc.EnableIPv6 {
			dirs = append(dirs, c2qtypes.Directive{Key: "IPv6", Values: []string{"true"}})
		}
		if nc.Attachable {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: name,
				Field:   "networks." + name + ".attachable",
				Message: "swarm overlay only",
			})
		}
		for _, pool := range nc.Ipam.Config {
			if len(pool.AuxiliaryAddresses) > 0 {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "networks." + name + ".ipam.config.aux_addresses",
					Message: "no quadlet directive",
				})
				break
			}
		}
		for _, k := range sortedKeys(nc.Labels) {
			if cfg.PodmanVersion.AtLeast(5, 6) {
				dirs = append(dirs, c2qtypes.Directive{Key: "Label", Values: []string{fmt.Sprintf("%s=%s", k, nc.Labels[k])}})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "networks." + name + ".labels",
					Message: "requires podman >= 5.6.0",
					Since:   "5.6.0",
				})
				break
			}
		}

		units = append(units, c2qtypes.QuadletUnit{
			Type:     c2qtypes.UnitNetwork,
			Name:     name,
			Sections: []c2qtypes.Section{{Name: c2qtypes.SectionNetwork, Directives: dirs}},
		})
	}
	return units
}
