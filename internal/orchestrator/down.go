package orchestrator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
)

// Down stops all units, removes quadlet files, removes networks, and unregisters the project.
// If removeVolumes is true, also removes Podman volumes.
func (o *Orchestrator) Down(removeVolumes bool, dryRun bool) error {
	stateMgr, state, err := o.ensureProjectDeployed()
	if err != nil {
		return err
	}

	if dryRun {
		logger.Printf("Dry run: project '%s' — would:\n", o.projectName)
		logger.Printf("  Stop %d unit(s)\n", len(state.Files))
		for _, f := range state.Files {
			logger.Print("    " + filepath.Base(f))
		}
		logger.Print("  Remove quadlet files and unregister project")
		logger.Print("  Remove podman networks")
		if removeVolumes {
			logger.Print("  Remove podman volumes")
		}
		return nil
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	if err := o.stopUnits(dbusMgr, state.Files); err != nil {
		logger.Warn("Some units failed to stop: " + err.Error())
	}

	for _, f := range state.Files {
		if strings.HasSuffix(f, ".network") {
			unitName := NetworkFileToUnitName(f)
			logger.Print("Stopping unit: " + unitName)
			if err := dbusMgr.StopUnit(unitName); err != nil {
				logger.Warn("Failed to stop network unit " + unitName + ": " + err.Error())
			}
		}
	}

	for _, f := range state.Files {
		if strings.HasSuffix(f, ".volume") {
			unitName := VolumeFileToUnitName(f)
			logger.Print("Stopping unit: " + unitName)
			if err := dbusMgr.StopUnit(unitName); err != nil {
				logger.Warn("Failed to stop volume unit " + unitName + ": " + err.Error())
			}
		}
	}

	for _, f := range state.Files {
		if strings.HasSuffix(f, ".image") {
			unitName := ImageFileToUnitName(f)
			logger.Print("Stopping unit: " + unitName)
			if err := dbusMgr.StopUnit(unitName); err != nil {
				logger.Warn("Failed to stop image unit " + unitName + ": " + err.Error())
			}
		}
	}

	for _, f := range state.Files {
		if strings.HasSuffix(f, ".build") {
			unitName := BuildFileToUnitName(f)
			logger.Print("Stopping unit: " + unitName)
			if err := dbusMgr.StopUnit(unitName); err != nil {
				logger.Warn("Failed to stop build unit " + unitName + ": " + err.Error())
			}
		}
	}

	for _, f := range state.Files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			logger.Warn("Failed to remove file " + f + ": " + err.Error())
		}
	}

	if err := dbusMgr.ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	if err := deploy.RemoveNetworks(o.projectName); err != nil {
		logger.Error("failed to remove networks: " + err.Error())
	}

	if removeVolumes {
		if err := deploy.RemoveVolumes(o.projectName); err != nil {
			logger.Error("failed to remove volumes: " + err.Error())
		}
	}

	if err := stateMgr.UnregisterProject(o.projectName); err != nil {
		return fmt.Errorf("failed to unregister project: %w", err)
	}

	if secretsDir, err := resolveSecretsDir(o.projectName); err == nil {
		os.RemoveAll(secretsDir)
	}

	logger.Success("Successfully removed project: " + o.projectName)
	return nil
}

func (o *Orchestrator) stopUnits(dbusMgr deploy.SystemdClient, projectFiles []string) error {
	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".container") {
			unitName := ContainerFileToUnitName(f)
			logger.Print("Stopping unit: " + unitName)
			if err := dbusMgr.StopUnit(unitName); err != nil {
				return fmt.Errorf("failed to stop unit %s: %w", unitName, err)
			}
		}
	}

	return nil
}

func (o *Orchestrator) verifyUnitsStopped(dbusMgr deploy.SystemdClient, projectFiles []string) error {
	var activeUnits []string
	var errs []error
	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		unitName := ContainerFileToUnitName(f)
		units, err := dbusMgr.ListUnitsByNames([]string{unitName})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to check unit %s: %w", unitName, err))
			continue
		}
		if len(units) > 0 && units[0].ActiveState == "active" {
			activeUnits = append(activeUnits, unitName)
		}
	}

	if len(activeUnits) > 0 {
		return fmt.Errorf("units still active after stop: %s", strings.Join(activeUnits, ", "))
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to verify all units stopped: %w", errors.Join(errs...))
	}

	return nil
}
