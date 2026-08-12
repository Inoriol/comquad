package mapper

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func Unit(svc types.ServiceConfig) []c2qtypes.Directive {
	return dependsOn(svc.Name, svc.DependsOn)
}

func UnitService(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	return dependsOnHealth(svc.DependsOn, cfg)
}

func dependsOn(serviceName string, deps types.DependsOnConfig) []c2qtypes.Directive {
	if len(deps) == 0 {
		return nil
	}
	var requires, wants, after, bindsTo []string
	for name, dep := range deps {
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
	for name, dep := range deps {
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
