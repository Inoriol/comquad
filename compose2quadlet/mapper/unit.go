package mapper

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func Unit(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	dirs := dependsOn(svc.Name, svc.DependsOn)
	dirs = append(dirs, pseudoQuadletDeps(svc, cfg)...)
	_, unitExt := ExtractSystemdExtensions(svc, cfg)
	return MergeDirectives(dirs, unitExt)
}

func pseudoQuadletDeps(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	if !cfg.PseudoQuadlet {
		return nil
	}
	var after, requires []string
	if svc.Build != nil && !cfg.PodmanVersion.AtLeast(5, 2) {
		after = append(after, svc.Name+".cq-build")
		requires = append(requires, svc.Name+".cq-build")
	} else if svc.Image != "" && !cfg.PodmanVersion.AtLeast(4, 8) {
		after = append(after, svc.Name+".cq-image")
		requires = append(requires, svc.Name+".cq-image")
	}
	var dirs []c2qtypes.Directive
	if len(requires) > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "Requires", Values: requires})
	}
	if len(after) > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "After", Values: after})
	}
	return dirs
}

func UnitService(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	return dependsOnHealth(svc.DependsOn, cfg)
}

func dependsOn(serviceName string, deps types.DependsOnConfig) []c2qtypes.Directive {
	if len(deps) == 0 {
		return nil
	}
	var requires, wants, after, bindsTo []string
	for _, name := range sortedKeys(deps) {
		dep := deps[name]
		after = append(after, name+".container")
		if dep.Required {
			requires = append(requires, name+".container")
		} else {
			wants = append(wants, name+".container")
		}
		if dep.Restart {
			bindsTo = append(bindsTo, name+".container")
		}
	}
	var dirs []c2qtypes.Directive
	if len(requires) > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "Requires", Values: requires})
	}
	if len(wants) > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "Wants", Values: wants})
	}
	if len(after) > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "After", Values: after})
	}
	if len(bindsTo) > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "BindsTo", Values: bindsTo})
	}
	return dirs
}

func dependsOnHealth(deps types.DependsOnConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive
	for _, name := range sortedKeys(deps) {
		dep := deps[name]
		if dep.Condition != "service_healthy" {
			continue
		}
		containerName := name
		if cfg.ProjectName != "" {
			containerName = cfg.ProjectName + "-" + name
		}
		cmd := fmt.Sprintf("/bin/sh -c 'while ! /usr/bin/podman healthcheck run %s; do sleep 1; done'", containerName)
		dirs = append(dirs, c2qtypes.Directive{Key: "ExecStartPre", Values: []string{cmd}})
	}
	return dirs
}
