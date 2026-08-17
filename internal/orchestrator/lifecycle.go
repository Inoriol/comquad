package orchestrator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
)

// resolveUnits resolves unit names from the project state.
// If services is empty, all container units are returned.
// If services is non-empty, only matching container units are returned.
func (o *Orchestrator) resolveUnits(services []string) ([]string, error) {
	_, state, err := o.ensureProjectDeployed()
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		var units []string
		for _, f := range state.Files {
			if strings.HasSuffix(f, ".container") || strings.HasSuffix(f, ".image") || strings.HasSuffix(f, ".build") {
				units = append(units, fileToUnitName(f))
			}
		}
		return units, nil
	}

	seen := make(map[string]struct{})
	var units []string
	for _, svc := range services {
		matches := MatchAllContainers(o.projectName, state, svc)
		// Also match image and build files
		for _, f := range state.Files {
			base := filepath.Base(f)
			var nameWithoutExt, ext string
			if strings.HasSuffix(base, ".image") {
				nameWithoutExt = strings.TrimSuffix(base, ".image")
				ext = ".image"
			} else if strings.HasSuffix(base, ".build") {
				nameWithoutExt = strings.TrimSuffix(base, ".build")
				ext = ".build"
			} else {
				continue
			}
			servicePrefix := "cq-" + o.projectName + "-"
			if base == svc ||
				nameWithoutExt == svc ||
				svc == nameWithoutExt+"-"+ext[1:]+".service" ||
				strings.TrimSuffix(svc, ".service") == nameWithoutExt+"-"+ext[1:] ||
				strings.TrimPrefix(nameWithoutExt, servicePrefix) == svc ||
				strings.TrimPrefix(nameWithoutExt, "cq-") == svc {
				matches = append(matches, f)
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no units found matching service '%s' for project '%s'", svc, o.projectName)
		}
		for _, f := range matches {
			unitName := fileToUnitName(f)
			if _, exists := seen[unitName]; !exists {
				seen[unitName] = struct{}{}
				units = append(units, unitName)
			}
		}
	}

	return units, nil
}

// fileToUnitName converts a quadlet file path to its systemd unit name.
func fileToUnitName(f string) string {
	switch {
	case strings.HasSuffix(f, ".container"):
		return ContainerFileToUnitName(f)
	case strings.HasSuffix(f, ".network"):
		return NetworkFileToUnitName(f)
	case strings.HasSuffix(f, ".volume"):
		return VolumeFileToUnitName(f)
	case strings.HasSuffix(f, ".image"):
		return ImageFileToUnitName(f)
	case strings.HasSuffix(f, ".build"):
		return BuildFileToUnitName(f)
	default:
		return ""
	}
}

// Start starts all units for the project, or specific services if provided.
func (o *Orchestrator) Start(services []string, dryRun bool) error {
	units, err := o.resolveUnits(services)
	if err != nil {
		return err
	}

	if dryRun {
		logger.Printf("Dry run: would start %d unit(s):\n", len(units))
		for _, unitName := range units {
			logger.Print("  " + unitName)
		}
		return nil
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return err
	}
	defer dbusMgr.Close()

	for _, unitName := range units {
		logger.Print("Starting unit: " + unitName)
		if err := dbusMgr.StartUnit(unitName); err != nil {
			return fmt.Errorf("failed to start unit %s: %w", unitName, err)
		}
	}

	if len(services) == 0 {
		logger.Print("Successfully started project: " + o.projectName)
	} else {
		logger.Printf("Successfully started %d unit(s) for project: %s\n", len(units), o.projectName)
	}

	return nil
}

// Stop stops all units for the project, or specific services if provided.
func (o *Orchestrator) Stop(services []string, dryRun bool) error {
	units, err := o.resolveUnits(services)
	if err != nil {
		return err
	}

	if dryRun {
		logger.Printf("Dry run: would stop %d unit(s):\n", len(units))
		for _, unitName := range units {
			logger.Print("  " + unitName)
		}
		return nil
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return err
	}
	defer dbusMgr.Close()

	for _, unitName := range units {
		logger.Print("Stopping unit: " + unitName)
		if err := dbusMgr.StopUnit(unitName); err != nil {
			return fmt.Errorf("failed to stop unit %s: %w", unitName, err)
		}
	}

	// Verify units are actually stopped (reuse the existing D-Bus connection)
	if err := o.verifyUnitsStoppedByNames(dbusMgr, units); err != nil {
		return err
	}

	if len(services) == 0 {
		logger.Print("Successfully stopped project: " + o.projectName)
	} else {
		logger.Printf("Successfully stopped %d unit(s) for project: %s\n", len(units), o.projectName)
	}

	return nil
}

// Restart restarts all units for the project, or specific services if provided.
func (o *Orchestrator) Restart(services []string, dryRun bool) error {
	units, err := o.resolveUnits(services)
	if err != nil {
		return err
	}

	if dryRun {
		logger.Printf("Dry run: would restart %d unit(s):\n", len(units))
		for _, unitName := range units {
			logger.Print("  " + unitName)
		}
		return nil
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return err
	}
	defer dbusMgr.Close()

	for _, unitName := range units {
		logger.Print("Restarting unit: " + unitName)
		if err := dbusMgr.RestartUnit(unitName); err != nil {
			return fmt.Errorf("failed to restart unit %s: %w", unitName, err)
		}
	}

	if len(services) == 0 {
		logger.Print("Successfully restarted project: " + o.projectName)
	} else {
		logger.Printf("Successfully restarted %d unit(s) for project: %s\n", len(units), o.projectName)
	}

	return nil
}

// verifyUnitsStoppedByNames verifies that the given unit names are no longer active.
func (o *Orchestrator) verifyUnitsStoppedByNames(dbusMgr deploy.SystemdClient, unitNames []string) error {
	units, err := dbusMgr.ListUnitsByNames(unitNames)
	if err != nil {
		return fmt.Errorf("failed to list units: %w", err)
	}

	var activeUnits []string
	for _, u := range units {
		if u.ActiveState == "active" {
			activeUnits = append(activeUnits, u.Name)
		}
	}

	if len(activeUnits) > 0 {
		return fmt.Errorf("units still active after stop: %s", strings.Join(activeUnits, ", "))
	}

	return nil
}
