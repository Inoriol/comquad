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

func (o *Orchestrator) resolveSystemdUnitDir() (string, error) {
	resolver := deploy.NewTargetDirResolver()
	unitDir, err := resolver.GetSystemdUnitPath()
	if err != nil {
		return "", fmt.Errorf("failed to resolve systemd unit path: %w", err)
	}
	return unitDir, nil
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
		if strings.HasSuffix(base, ".cq-secret") {
			name := strings.TrimPrefix(base, prefix)
			name = strings.TrimSuffix(name, ".cq-secret")
			resources.Secrets = append(resources.Secrets, name)
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

	// Start secret services first
	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".cq-secret") {
			continue
		}
		unitName := CQSecretFileToUnitName(f)
		if !output.IsJSONMode() {
			logger.Action("Starting secret unit: " + unitName)
		}

		if err := dbusMgr.WaitForUnit(unitName, startUnitWaitTime); err != nil {
			return fmt.Errorf("unit %s did not appear after daemon-reload: %w", unitName, err)
		}

		// Always restart secret services to pick up any value changes
		if err := dbusMgr.RestartUnit(unitName); err != nil {
			return fmt.Errorf("failed to restart unit %s: %w", unitName, err)
		}
	}

	// Start container units
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

func (o *Orchestrator) copyPseudoQuadlets(projectFiles []string) error {
	systemdUnitDir, err := o.resolveSystemdUnitDir()
	if err != nil {
		return fmt.Errorf("failed to resolve systemd unit directory: %w", err)
	}

	if err := os.MkdirAll(systemdUnitDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd unit directory: %w", err)
	}

	hasSecrets := false
	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".cq-secret") {
			hasSecrets = true
			break
		}
	}

	if hasSecrets {
		secretDir := "/run/comquad/secrets"
		if err := os.MkdirAll(secretDir, 0700); err != nil {
			return fmt.Errorf("failed to create secret directory: %w", err)
		}
	}

	for _, f := range projectFiles {
		base := filepath.Base(f)
		var serviceName string
		switch {
		case strings.HasSuffix(base, ".cq-secret"):
			serviceName = CQSecretFileToUnitName(f)
		case strings.HasSuffix(base, ".cq-image"):
			serviceName = CQImageFileToUnitName(f)
		case strings.HasSuffix(base, ".cq-build"):
			serviceName = CQBuildFileToUnitName(f)
		default:
			continue
		}
		servicePath := filepath.Join(systemdUnitDir, serviceName)
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read pseudo-quadlet %s: %w", f, err)
		}

		if strings.HasSuffix(base, ".cq-secret") {
			content, err = o.processSecretFile(content, base)
			if err != nil {
				return fmt.Errorf("failed to process secret file %s: %w", base, err)
			}
		}

		if err := os.WriteFile(servicePath, content, 0644); err != nil {
			return fmt.Errorf("failed to write service file %s: %w", servicePath, err)
		}
	}
	return nil
}

func (o *Orchestrator) processSecretFile(content []byte, filename string) ([]byte, error) {
	lines := strings.Split(string(content), "\n")
	var result []string

	for _, line := range lines {
		if strings.HasPrefix(line, "ExecStart=") && strings.Contains(line, "/run/comquad/secrets/") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				secretPath := strings.TrimPrefix(parts[1], "/bin/sh -c 'podman secret create ")
				secretPath = strings.TrimPrefix(secretPath, "--replace ")
				secretPath = strings.Split(secretPath, " ")[1]
				secretPath = strings.TrimSuffix(secretPath, "'")
				
				envVarName := extractEnvVarNameFromServiceUnit(string(content))
				if envVarName != "" {
					envValue := os.Getenv(envVarName)
					if envValue != "" {
						if err := os.WriteFile(secretPath, []byte(envValue), 0600); err != nil {
							return nil, fmt.Errorf("failed to write secret file: %w", err)
						}
					}
				}
			}
		}
		result = append(result, line)
	}

	return []byte(strings.Join(result, "\n")), nil
}

func extractEnvVarNameFromServiceUnit(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "COMQUAD_SECRET_ENV=") {
			parts := strings.SplitN(line, "COMQUAD_SECRET_ENV=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func (o *Orchestrator) removePseudoQuadletServices(projectFiles []string) error {
	systemdUnitDir, err := o.resolveSystemdUnitDir()
	if err != nil {
		return fmt.Errorf("failed to resolve systemd unit directory: %w", err)
	}

	for _, f := range projectFiles {
		base := filepath.Base(f)
		var serviceName string
		switch {
		case strings.HasSuffix(base, ".cq-secret"):
			serviceName = CQSecretFileToUnitName(f)
		case strings.HasSuffix(base, ".cq-image"):
			serviceName = CQImageFileToUnitName(f)
		case strings.HasSuffix(base, ".cq-build"):
			serviceName = CQBuildFileToUnitName(f)
		default:
			continue
		}
		servicePath := filepath.Join(systemdUnitDir, serviceName)
		if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove service file %s: %w", servicePath, err)
		}
	}
	return nil
}

func stringSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}
