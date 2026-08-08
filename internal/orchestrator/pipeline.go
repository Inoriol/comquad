package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Inoriol/comquad/internal/cooker"
	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/graft"
	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/preprocess"
	"github.com/Inoriol/comquad/internal/transpile"
)

func (o *Orchestrator) resolveTargetDir() (string, error) {
	resolver := deploy.NewTargetDirResolver()
	targetDir, err := resolver.GetSystemdPath()
	if err != nil {
		return "", fmt.Errorf("failed to resolve systemd path: %w", err)
	}
	return targetDir, nil
}

func (o *Orchestrator) preprocess(composeData []byte) ([]byte, error) {
	engine := preprocess.NewEngine(o.projectName, o.cwd)
	processed, err := engine.Process(composeData)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}
	return processed, nil
}

func (o *Orchestrator) transpile(processedYaml []byte, tempDir string) error {
	podlet, err := transpile.NewPodletRunner(tempDir)
	if err != nil {
		return err
	}
	if err := podlet.Transpile(processedYaml); err != nil {
		return fmt.Errorf("transpilation failed: %w", err)
	}
	return nil
}

func (o *Orchestrator) cook(tempDir, targetDir string, isRootless bool) (map[string]string, error) {
	portOffset := 0
	if isRootless {
		portOffset = defaultPortOffset
		if envOffset := os.Getenv("ROOTLESS_PORT_OFFSET"); envOffset != "" {
			if parsed, err := strconv.Atoi(envOffset); err == nil && parsed > 0 {
				portOffset = parsed
			}
		}
	}

	selinuxEnabled := preprocess.IsSELinuxEnabled()
	if selinuxEnabled {
		logger.Action(fmt.Sprintf("SELinux detected (%s), adding :z labels to all volumes", preprocess.SELinuxMode()))
	}

	cookerEngine := cooker.NewCooker(tempDir, targetDir, o.projectName, isRootless, portOffset, selinuxEnabled)
	result, err := cookerEngine.Cook()
	if err != nil {
		return nil, fmt.Errorf("cooking failed: %w", err)
	}
	return result.FileContents, nil
}

func (o *Orchestrator) graft(
	fileContents map[string]string,
	services map[string]preprocess.ServiceImageSpec,
	secretDefs map[string]preprocess.SecretDef,
	serviceSecretRefs preprocess.ServiceSecretRefs,
	secretsDir string,
	dryRun bool,
) map[string]string {
	grafter := graft.Grafter{
		ProjectName:    o.projectName,
		SecretsDir:     secretsDir,
		DryRun:         dryRun,
		SELinuxEnabled: preprocess.IsSELinuxEnabled(),
	}
	return grafter.Process(fileContents, services, secretDefs, serviceSecretRefs)
}

func (o *Orchestrator) collectProjectFiles(targetDir string) ([]string, error) {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read target directory: %w", err)
	}

	var projectFiles []string
	prefix := "cq-" + o.projectName
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

func (o *Orchestrator) startUnits(projectFiles []string) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	var reloadFiles []string
	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".container") {
			reloadFiles = append(reloadFiles, f)
		}
	}

	if err := dbusMgr.ReloadDaemon(reloadFiles...); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".image") {
			unitName := ImageFileToUnitName(f)
			logger.Action("Starting unit: " + unitName)
			if err := dbusMgr.WaitForUnit(unitName, startUnitWaitTime); err != nil {
				logger.Warn(fmt.Sprintf("image unit %s not produced by quadlet generator, skipping: %v", unitName, err))
				continue
			}
			if err := dbusMgr.StartUnit(unitName); err != nil {
				logger.Warn(fmt.Sprintf("failed to start image unit %s: %v", unitName, err))
			}
		}
	}

	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".container") {
			unitName := ContainerFileToUnitName(f)
			logger.Action("Starting unit: " + unitName)

			if err := dbusMgr.WaitForUnit(unitName, startUnitWaitTime); err != nil {
				return fmt.Errorf("unit %s did not appear after daemon-reload: %w", unitName, err)
			}

			if err := dbusMgr.StartUnit(unitName); err != nil {
				return fmt.Errorf("failed to start unit %s: %w", unitName, err)
			}
		}
	}

	return nil
}
