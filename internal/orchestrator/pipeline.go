package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/output"
	"github.com/Inoriol/comquad/internal/reconcile"
)

func (o *Orchestrator) resolveTargetDir() (string, error) {
	resolver := deploy.NewTargetDirResolver()
	targetDir, err := resolver.GetSystemdPath()
	if err != nil {
		return "", fmt.Errorf("failed to resolve systemd path: %w", err)
	}
	return targetDir, nil
}

func (o *Orchestrator) collectProjectFiles(targetDir string) ([]string, error) {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read target directory: %w", err)
	}

	var projectFiles []string
	prefix := "cq-" + o.projectName + "-"
	for _, f := range entries {
		if strings.HasPrefix(f.Name(), prefix) {
			projectFiles = append(projectFiles, filepath.Join(targetDir, f.Name()))
		}
	}

	return projectFiles, nil
}

func (o *Orchestrator) registerState(projectFiles []string) error {
	stateMgr, err := o.newState()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	resources := &deploy.ResourceInfo{}
	prefix := "cq-" + o.projectName + "-"
	for _, f := range projectFiles {
		base := filepath.Base(f)
		if strings.HasSuffix(base, ".container") {
			name := strings.TrimPrefix(base, prefix)
			name = strings.TrimSuffix(name, ".container")
			resources.Containers = append(resources.Containers, o.projectName+"-"+name)
		}
		if strings.HasSuffix(base, ".network") {
			name := strings.TrimPrefix(base, prefix)
			name = strings.TrimSuffix(name, ".network")
			resources.Networks = append(resources.Networks, name)
		}
		if strings.HasSuffix(base, ".volume") {
			name := strings.TrimPrefix(base, prefix)
			name = strings.TrimSuffix(name, ".volume")
			resources.Volumes = append(resources.Volumes, name)
		}
		if strings.HasSuffix(base, ".image") {
			name := strings.TrimPrefix(base, prefix)
			name = strings.TrimSuffix(name, ".image")
			resources.Images = append(resources.Images, name)
		}
		if strings.HasSuffix(base, ".build") {
			name := strings.TrimPrefix(base, prefix)
			name = strings.TrimSuffix(name, ".build")
			resources.Builds = append(resources.Builds, name)
		}
	}

	return stateMgr.RegisterProject(deploy.ProjectState{
		ProjectName: o.projectName,
		SourcePath:  o.cwd,
		Files:       projectFiles,
		Resources:   resources,
	})
}

func (o *Orchestrator) prepareUnits(projectFiles []string, res reconcile.Result) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	for _, f := range res.Removed {
		unitName := fileToUnitName(f)
		if unitName == "" {
			continue
		}
		if !output.IsJSONMode() {
			logger.Action("Stopping removed unit: " + unitName)
		}
		if err := dbusMgr.StopUnit(unitName); err != nil {
			logger.Warn(fmt.Sprintf("failed to stop removed unit %s: %v", unitName, err))
		}
	}

	var reloadFiles []string
	reloadFiles = append(reloadFiles, res.Created...)
	reloadFiles = append(reloadFiles, res.Changed...)

	if err := dbusMgr.ReloadDaemon(reloadFiles...); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	return nil
}

func (o *Orchestrator) startContainers(projectFiles []string, res reconcile.Result) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	changed := stringSet(res.Changed)

	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		unitName := ContainerFileToUnitName(f)
		if !output.IsJSONMode() {
			logger.Action("Starting unit: " + unitName)
		}

		if err := dbusMgr.WaitForUnit(unitName, startUnitWaitTime); err != nil {
			return fmt.Errorf("unit %s did not appear after daemon-reload: %w", unitName, err)
		}

		if changed[f] {
			if err := dbusMgr.RestartUnit(unitName); err != nil {
				return fmt.Errorf("failed to restart unit %s: %w", unitName, err)
			}
		} else {
			if err := dbusMgr.StartUnit(unitName); err != nil {
				return fmt.Errorf("failed to start unit %s: %w", unitName, err)
			}
		}
	}

	return nil
}

func (o *Orchestrator) reportReconcile(res reconcile.Result) {
	if output.IsJSONMode() {
		return
	}
	for _, c := range res.Conflicts {
		logger.Warn(fmt.Sprintf("conflict in %s [%s] %s: keeping your edit %q, generated %q", c.Unit, c.Section, c.Key, c.User, c.Generated))
	}
	for _, f := range res.NoBaseline {
		logger.Warn(fmt.Sprintf("no baseline for %s — overwriting with generated content", filepath.Base(f)))
	}
}

func stringSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}
