package opinionated

import c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"

func Apply(units []c2qtypes.QuadletUnit, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	units = ApplyPrefix(units, cfg)
	units = ApplyReferences(units, cfg)
	units = ApplyContainerName(units, cfg)
	units = ApplyDefaultNetwork(units, cfg)
	units = ApplyNetworkAliases(units, cfg)
	units = ApplySELinux(units, cfg)
	units = ApplyLabels(units, cfg)
	units = ApplyPortOffset(units, cfg)
	units = ApplyAutoUpdate(units, cfg)
	units = ApplyInstallSection(units, cfg)
	return units
}
