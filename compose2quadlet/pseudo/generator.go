package pseudo

import (
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func ConvertPseudoUnits(units []c2qtypes.QuadletUnit) []c2qtypes.QuadletUnit {
	var result []c2qtypes.QuadletUnit

	for _, u := range units {
		switch u.Type {
		case c2qtypes.UnitCQSecret:
			result = append(result, convertSecret(u))
		case c2qtypes.UnitCQImage:
			result = append(result, convertImage(u))
		case c2qtypes.UnitCQBuild:
			result = append(result, convertBuild(u))
		default:
			result = append(result, u)
		}
	}

	return result
}
