package pseudo

import (
	"fmt"
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func convertBuild(unit c2qtypes.QuadletUnit) c2qtypes.QuadletUnit {
	var buildSection *c2qtypes.Section
	for i := range unit.Sections {
		if unit.Sections[i].Name == c2qtypes.SectionBuild {
			buildSection = &unit.Sections[i]
			break
		}
	}

	var workDir, dockerfile, imageTag string
	var buildArgs []string
	var noCache bool

	if buildSection != nil {
		for _, d := range buildSection.Directives {
			switch d.Key {
			case "SetWorkingDirectory":
				if len(d.Values) > 0 {
					workDir = d.Values[0]
				}
			case "File":
				if len(d.Values) > 0 {
					dockerfile = d.Values[0]
				}
			case "ImageTag":
				if len(d.Values) > 0 {
					if imageTag == "" {
						imageTag = d.Values[0]
					}
				}
			case "BuildArg":
				for _, v := range d.Values {
					buildArgs = append(buildArgs, "--build-arg", v)
				}
			case "PodmanArgs":
				for _, v := range d.Values {
					if strings.Contains(v, "--no-cache") {
						noCache = true
					}
				}
			}
		}
	}

	var execArgs []string
	execArgs = append(execArgs, "/usr/bin/podman", "build")

	if dockerfile != "" {
		execArgs = append(execArgs, "-f", dockerfile)
	}

	if imageTag != "" {
		execArgs = append(execArgs, "-t", imageTag)
	}

	if noCache {
		execArgs = append(execArgs, "--no-cache")
	}

	for _, arg := range buildArgs {
		execArgs = append(execArgs, arg)
	}

	execArgs = append(execArgs, ".")

	serviceDirs := []c2qtypes.Directive{
		{Key: "Type", Values: []string{"oneshot"}},
		{Key: "RemainAfterExit", Values: []string{"yes"}},
	}

	if workDir != "" {
		serviceDirs = append(serviceDirs, c2qtypes.Directive{
			Key:    "WorkingDirectory",
			Values: []string{workDir},
		})
	}

	serviceDirs = append(serviceDirs, c2qtypes.Directive{
		Key:    "ExecStart",
		Values: []string{strings.Join(execArgs, " ")},
	})

	unitDirs := []c2qtypes.Directive{
		{Key: "Description", Values: []string{fmt.Sprintf("comquad image build: %s", unit.Name)}},
	}

	installDirs := []c2qtypes.Directive{
		{Key: "WantedBy", Values: []string{"default.target"}},
	}

	return c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQBuild,
		Name: unit.Name,
		Sections: []c2qtypes.Section{
			{Name: c2qtypes.SectionUnit, Directives: unitDirs},
			{Name: c2qtypes.SectionService, Directives: serviceDirs},
			{Name: c2qtypes.SectionInstall, Directives: installDirs},
		},
	}
}
