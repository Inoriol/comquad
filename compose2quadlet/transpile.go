package compose2quadlet

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/Inoriol/comquad/compose2quadlet/mapper"
	"github.com/Inoriol/comquad/compose2quadlet/opinionated"
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
)

func Transpile(project *types.Project, opts ...TranspileOption) ([]QuadletUnit, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.ExternalNetworks == nil {
		cfg.ExternalNetworks = make(map[string]string)
	}
	for name, network := range project.Networks {
		if network.External {
			actual := network.Name
			if actual == "" {
				actual = name
			}
			cfg.ExternalNetworks[name] = actual
		}
	}
	if cfg.ExternalVolumes == nil {
		cfg.ExternalVolumes = make(map[string]string)
	}
	for name, volume := range project.Volumes {
		if volume.External {
			actual := volume.Name
			if actual == "" {
				actual = name
			}
			cfg.ExternalVolumes[name] = actual
		}
	}

	cfg.SharedVolumes = analyzeSharedVolumes(project.Services)

	cfg.JobNames = make(map[string]bool)
	for name := range project.Jobs {
		cfg.JobNames[name] = true
	}

	var units []QuadletUnit
	secretUnitMap := make(map[string]QuadletUnit)

	for name := range project.Services {
		svc := project.Services[name]

		secretDirs, secretUnits := mapper.PremapSecrets(&svc, project.Secrets, project.Configs, cfg)
		project.Services[name] = svc

		for _, su := range secretUnits {
			if _, exists := secretUnitMap[su.Name]; !exists {
				secretUnitMap[su.Name] = su
			}
		}

		var sections []Section

		unitDirs := mapper.Unit(svc, cfg)
		if len(unitDirs) > 0 {
			sections = append(sections, Section{Name: SectionUnit, Directives: unitDirs})
		}

		containerDirs := mapper.Container(svc, cfg)
		hcDirs := mapper.Healthcheck(svc, cfg)
		containerDirs = append(containerDirs, hcDirs...)
		containerDirs = append(containerDirs, secretDirs...)

		if len(containerDirs) > 0 {
			sections = append(sections, Section{Name: SectionContainer, Directives: containerDirs})
		}

		serviceDirs := mapper.Service(svc, cfg)
		depServiceDirs := mapper.UnitService(svc, cfg)
		serviceDirs = append(serviceDirs, depServiceDirs...)
		if len(serviceDirs) > 0 {
			sections = append(sections, Section{Name: SectionService, Directives: serviceDirs})
		}

		if len(sections) == 0 {
			continue
		}

		units = append(units, QuadletUnit{
			Type:     UnitContainer,
			Name:     name,
			Sections: sections,
		})
	}

	for _, su := range secretUnitMap {
		units = append(units, su)
	}

	imagUnits := mapper.Images(project.Services, cfg)
	units = append(units, imagUnits...)

	buildUnits := mapper.Builds(project.Services, cfg)
	units = append(units, buildUnits...)

	netUnits := mapper.Networks(project.Networks, cfg)
	units = append(units, netUnits...)

	volUnits := mapper.Volumes(project.Volumes, cfg)
	units = append(units, volUnits...)

	for name, job := range project.Jobs {
		var sections []Section

		unitDirs := mapper.JobUnit(job, cfg)
		if len(unitDirs) > 0 {
			sections = append(sections, Section{Name: SectionUnit, Directives: unitDirs})
		}

		containerDirs := mapper.JobContainer(job, cfg)
		if len(containerDirs) > 0 {
			sections = append(sections, Section{Name: SectionContainer, Directives: containerDirs})
		}

		if len(sections) == 0 {
			continue
		}

		units = append(units, QuadletUnit{
			Type:     UnitContainer,
			Name:     name,
			Sections: sections,
		})
	}

	jobImageUnits := mapper.JobImages(project.Jobs, cfg)
	units = append(units, jobImageUnits...)

	jobBuildUnits := mapper.JobBuilds(project.Jobs, cfg)
	units = append(units, jobBuildUnits...)

	timerUnits := mapper.Timers(project.Jobs, cfg)
	units = append(units, timerUnits...)

	for _, w := range cfg.Warnings {
		if w.Level == WarningFatal {
			return nil, errors.New(w.Message)
		}
	}

	units = opinionated.Apply(units, cfg)

	return units, nil
}

func TranspileFile(composePath string, opts ...TranspileOption) ([]QuadletUnit, error) {
	project, err := LoadProject(composePath)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(composePath)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(absPath)

	opts = append(opts,
		WithWorkingDirectory(dir),
	)

	return Transpile(project, opts...)
}

func LoadProject(composePath string) (*types.Project, error) {
	absPath, err := filepath.Abs(composePath)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(absPath)

	projectOpts, err := cli.NewProjectOptions(
		[]string{absPath},
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithWorkingDirectory(dir),
	)
	if err != nil {
		return nil, err
	}

	return projectOpts.LoadProject(context.Background())
}

func analyzeSharedVolumes(services types.Services) map[string]bool {
	namedVolRefs := make(map[string]int)
	bindMountRefs := make(map[string]int)

	for _, svc := range services {
		seenNamed := make(map[string]bool)
		seenBind := make(map[string]bool)
		for _, v := range svc.Volumes {
			if v.Type == types.VolumeTypeBind {
				source := v.Source
				if !seenBind[source] {
					bindMountRefs[source]++
					seenBind[source] = true
				}
			} else if v.Type != types.VolumeTypeTmpfs {
				name := v.Source
				if !seenNamed[name] {
					namedVolRefs[name]++
					seenNamed[name] = true
				}
			}
		}
	}

	shared := make(map[string]bool)
	for name, count := range namedVolRefs {
		if count > 1 {
			shared[name] = true
		}
	}
	for path, count := range bindMountRefs {
		if count > 1 {
			shared[path] = true
		}
	}
	return shared
}
