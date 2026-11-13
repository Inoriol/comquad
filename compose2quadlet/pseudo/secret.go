package pseudo

import (
	"fmt"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func convertSecret(unit c2qtypes.QuadletUnit) c2qtypes.QuadletUnit {
	var secretSection *c2qtypes.Section
	for i := range unit.Sections {
		if unit.Sections[i].Name == c2qtypes.SectionSecret {
			secretSection = &unit.Sections[i]
			break
		}
	}

	serviceDirs := []c2qtypes.Directive{
		{Key: "Type", Values: []string{"oneshot"}},
		{Key: "RemainAfterExit", Values: []string{"yes"}},
	}

	if secretSection != nil {
		var sourceType, sourceValue string
		for _, d := range secretSection.Directives {
			switch d.Key {
			case "Source":
				if len(d.Values) > 0 {
					sourceType = d.Values[0]
				}
			case "Environment":
				if len(d.Values) > 0 {
					sourceValue = d.Values[0]
				}
			case "Path":
				if len(d.Values) > 0 {
					sourceValue = d.Values[0]
				}
			}
		}

		switch sourceType {
		case "environment":
			secretFile := "/run/comquad/secrets/" + unit.Name + ".env"
			execStart := fmt.Sprintf("/bin/sh -c 'podman secret create --replace %s %s'", unit.Name, secretFile)
			serviceDirs = append(serviceDirs, c2qtypes.Directive{
				Key:    "ExecStart",
				Values: []string{execStart},
			})
			serviceDirs = append(serviceDirs, c2qtypes.Directive{
				Key:    "ExecStop",
				Values: []string{"-/usr/bin/podman secret rm " + unit.Name},
			})
			serviceDirs = append(serviceDirs, c2qtypes.Directive{
				Key:    "Environment",
				Values: []string{"COMQUAD_SECRET_ENV=" + sourceValue},
			})
		case "file":
			serviceDirs = append(serviceDirs, c2qtypes.Directive{
				Key:    "ExecStart",
				Values: []string{fmt.Sprintf("/usr/bin/podman secret create --replace %s %s", unit.Name, sourceValue)},
			})
			serviceDirs = append(serviceDirs, c2qtypes.Directive{
				Key:    "ExecStop",
				Values: []string{"-/usr/bin/podman secret rm " + unit.Name},
			})
		}
	}

	unitDirs := []c2qtypes.Directive{
		{Key: "Description", Values: []string{fmt.Sprintf("comquad secret: %s", unit.Name)}},
	}

	installDirs := []c2qtypes.Directive{
		{Key: "WantedBy", Values: []string{"default.target"}},
	}

	return c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQSecret,
		Name: unit.Name,
		Sections: []c2qtypes.Section{
			{Name: c2qtypes.SectionUnit, Directives: unitDirs},
			{Name: c2qtypes.SectionService, Directives: serviceDirs},
			{Name: c2qtypes.SectionInstall, Directives: installDirs},
		},
	}
}
