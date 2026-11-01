package pseudo

import (
	"fmt"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func convertImage(unit c2qtypes.QuadletUnit) c2qtypes.QuadletUnit {
	var imageSection *c2qtypes.Section
	for i := range unit.Sections {
		if unit.Sections[i].Name == c2qtypes.SectionImage {
			imageSection = &unit.Sections[i]
			break
		}
	}

	var imageName string
	if imageSection != nil {
		for _, d := range imageSection.Directives {
			if d.Key == "Image" && len(d.Values) > 0 {
				imageName = d.Values[0]
				break
			}
		}
	}

	serviceDirs := []c2qtypes.Directive{
		{Key: "Type", Values: []string{"oneshot"}},
		{Key: "RemainAfterExit", Values: []string{"yes"}},
	}

	if imageName != "" {
		serviceDirs = append(serviceDirs, c2qtypes.Directive{
			Key:    "ExecStart",
			Values: []string{fmt.Sprintf("/usr/bin/podman pull %s", imageName)},
		})
	}

	unitDirs := []c2qtypes.Directive{
		{Key: "Description", Values: []string{fmt.Sprintf("comquad image pull: %s", unit.Name)}},
	}

	installDirs := []c2qtypes.Directive{
		{Key: "WantedBy", Values: []string{"default.target"}},
	}

	return c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQImage,
		Name: unit.Name,
		Sections: []c2qtypes.Section{
			{Name: c2qtypes.SectionUnit, Directives: unitDirs},
			{Name: c2qtypes.SectionService, Directives: serviceDirs},
			{Name: c2qtypes.SectionInstall, Directives: installDirs},
		},
	}
}
