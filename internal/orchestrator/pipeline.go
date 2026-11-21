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

func (o *Orchestrator) collectProjectFiles(quadletDir, systemdUnitDir string) ([]string, error) {
	var projectFiles []string
	prefix := "cq-" + o.projectName + "-"
	
	// Collect from quadlet directory
	entries, err := os.ReadDir(quadletDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read quadlet directory: %w", err)
	}
	for _, f := range entries {
		if strings.HasPrefix(f.Name(), prefix) {
			projectFiles = append(projectFiles, filepath.Join(quadletDir, f.Name()))
		}
	}
	
	// Collect timer files from systemd unit directory
	entries, err = os.ReadDir(systemdUnitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read systemd unit directory: %w", err)
	}
	for _, f := range entries {
		if strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ".timer") {
			projectFiles = append(projectFiles, filepath.Join(systemdUnitDir, f.Name()))
		}
	}

	return projectFiles, nil
}

func (o *Orchestrator) registerState(projectFiles []string, autoNSRange *deploy.AutoNSRange) error {
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
		if strings.HasSuffix(base, ".timer") {
			name := strings.TrimPrefix(base, prefix)
			name = strings.TrimSuffix(name, ".timer")
			resources.Timers = append(resources.Timers, name)
		}
	}

	return stateMgr.RegisterProject(deploy.ProjectState{
		ProjectName: o.projectName,
		SourcePath:  o.cwd,
		Files:       projectFiles,
		Resources:   resources,
		AutoNSRange: autoNSRange,
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

func (o *Orchestrator) startContainers(projectFiles []string, res reconcile.Result, anchorService string) error {
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

	// Start anchor container first if AutoNS is enabled
	if anchorService != "" {
		prefix := "cq-" + o.projectName + "-"
		anchorFileName := prefix + anchorService + ".container"
		for _, f := range projectFiles {
			if filepath.Base(f) == anchorFileName {
				unitName := ContainerFileToUnitName(f)
				if !output.IsJSONMode() {
					logger.Action("Starting anchor unit: " + unitName)
				}

				if err := dbusMgr.WaitForUnit(unitName, startUnitWaitTime); err != nil {
					return fmt.Errorf("unit %s did not appear after daemon-reload: %w", unitName, err)
				}

				if changed[f] {
					if err := dbusMgr.RestartUnit(unitName); err != nil {
						return fmt.Errorf("failed to restart anchor unit %s: %w", unitName, err)
					}
				} else {
					if err := dbusMgr.StartUnit(unitName); err != nil {
						return fmt.Errorf("failed to start anchor unit %s: %w", unitName, err)
					}
				}
				break
			}
		}
	}

	// Start service container units (skip jobs and anchor if already started)
	composeProject, _ := loadComposeProject(o.resolveComposeFile())
	jobNames := make(map[string]bool)
	if composeProject != nil {
		for name := range composeProject.Jobs {
			jobNames[name] = true
		}
	}
	prefix := "cq-" + o.projectName + "-"

	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		if anchorService != "" && filepath.Base(f) == "cq-"+o.projectName+"-"+anchorService+".container" {
			continue
		}

		containerName := strings.TrimPrefix(filepath.Base(f), prefix)
		containerName = strings.TrimSuffix(containerName, ".container")
		if jobNames[containerName] {
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

	// Start and enable timer units
	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".timer") {
			continue
		}
		unitName := TimerFileToUnitName(f)
		if !output.IsJSONMode() {
			logger.Action("Enabling and starting timer: " + unitName)
		}

		if err := dbusMgr.EnableUnit(unitName); err != nil {
			logger.Warn(fmt.Sprintf("failed to enable timer %s: %v", unitName, err))
		}

		if err := dbusMgr.StartUnit(unitName); err != nil {
			return fmt.Errorf("failed to start timer %s: %w", unitName, err)
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

	var runtimeSecretsDir string
	if hasSecrets {
		runtimeSecretsDir, err = deploy.ResolveRuntimeSecretsDir()
		if err != nil {
			return fmt.Errorf("failed to resolve runtime secrets directory: %w", err)
		}
		if err := os.MkdirAll(runtimeSecretsDir, 0700); err != nil {
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
			content, err = o.processSecretFile(content, base, runtimeSecretsDir)
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

func (o *Orchestrator) processSecretFile(content []byte, filename string, runtimeSecretsDir string) ([]byte, error) {
	lines := strings.Split(string(content), "\n")
	var result []string

	for _, line := range lines {
		if strings.HasPrefix(line, "ExecStart=") && strings.Contains(line, runtimeSecretsDir+"/") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				execValue := parts[1]
				secretPath := extractSecretPath(execValue, runtimeSecretsDir)
				if secretPath != "" {
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
		}
		result = append(result, line)
	}

	return []byte(strings.Join(result, "\n")), nil
}

func extractSecretPath(execValue, runtimeSecretsDir string) string {
	idx := strings.Index(execValue, runtimeSecretsDir)
	if idx == -1 {
		return ""
	}
	rest := execValue[idx:]
	endIdx := strings.IndexAny(rest, " '")
	if endIdx == -1 {
		return rest
	}
	return rest[:endIdx]
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
