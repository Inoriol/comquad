package pseudo

import (
	"fmt"
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func convertSecret(unit c2qtypes.QuadletUnit, runtimeSecretsDir string) c2qtypes.QuadletUnit {
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
		var sourceType, sourceValue, driver, driverOpts string
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
			case "Driver":
				if len(d.Values) > 0 {
					driver = d.Values[0]
				}
			case "DriverOpts":
				if len(d.Values) > 0 {
					driverOpts = d.Values[0]
				}
			}
		}

		execStart := buildSecretExecStart(unit.Name, sourceType, sourceValue, driver, driverOpts, runtimeSecretsDir)
		serviceDirs = append(serviceDirs, c2qtypes.Directive{
			Key:    "ExecStart",
			Values: []string{execStart},
		})
		serviceDirs = append(serviceDirs, c2qtypes.Directive{
			Key:    "ExecStop",
			Values: []string{"-/usr/bin/podman secret rm " + unit.Name},
		})
		if sourceType == "environment" {
			serviceDirs = append(serviceDirs, c2qtypes.Directive{
				Key:    "Environment",
				Values: []string{"COMQUAD_SECRET_ENV=" + sourceValue},
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

func buildSecretExecStart(name, sourceType, sourceValue, driver, driverOpts, runtimeSecretsDir string) string {
	var parts []string
	parts = append(parts, "/usr/bin/podman secret create")
	if driver != "" {
		parts = append(parts, "--driver="+driver)
	}
	if driverOpts != "" {
		parts = append(parts, "--driver-opts="+driverOpts)
	}
	parts = append(parts, "--replace", name)

	switch sourceType {
	case "environment":
		secretFile := runtimeSecretsDir + "/" + name + ".env"
		return "/bin/sh -c '" + strings.Join(parts, " ") + " " + secretFile + "'"
	case "file":
		parts = append(parts, sourceValue)
		return strings.Join(parts, " ")
	case "driver":
		return "/bin/sh -c '" + strings.Join(parts, " ") + " -'"
	default:
		return strings.Join(parts, " ")
	}
}
