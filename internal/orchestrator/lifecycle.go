package orchestrator

import (
	"fmt"
	"strings"

	"comquad/internal/deploy"
)

// resolveUnits resolves unit names from the project state.
// If services is empty, all container units are returned.
// If services is non-empty, only matching container units are returned.
func (o *Orchestrator) resolveUnits(services []string) ([]string, error) {
	stateMgr, err := deploy.NewStateManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	state, exists := stateMgr.Projects[o.projectName]
	if !exists {
		return nil, fmt.Errorf("project %s is not deployed", o.projectName)
	}

	if len(services) == 0 {
		var units []string
		for _, f := range state.Files {
			if strings.HasSuffix(f, ".container") {
				units = append(units, containerFileToUnitName(f))
			}
		}
		return units, nil
	}

	seen := make(map[string]struct{})
	var units []string
	for _, svc := range services {
		matches := MatchContainers(o.projectName, state, svc)
		if len(matches) == 0 {
			return nil, fmt.Errorf("no units found matching service '%s' for project '%s'", svc, o.projectName)
		}
		for _, f := range matches {
			unitName := containerFileToUnitName(f)
			if _, exists := seen[unitName]; !exists {
				seen[unitName] = struct{}{}
				units = append(units, unitName)
			}
		}
	}

	return units, nil
}

// Start starts all units for the project, or specific services if provided.
func (o *Orchestrator) Start(services []string) error {
	units, err := o.resolveUnits(services)
	if err != nil {
		return err
	}

	dbusMgr, err := deploy.NewSystemdManager()
	if err != nil {
		return err
	}
	defer dbusMgr.Close()

	for _, unitName := range units {
		fmt.Printf("Starting unit: %s\n", unitName)
		if err := dbusMgr.StartUnit(unitName); err != nil {
			return fmt.Errorf("failed to start unit %s: %w", unitName, err)
		}
	}

	if len(services) == 0 {
		fmt.Printf("Successfully started project: %s\n", o.projectName)
	} else {
		fmt.Printf("Successfully started %d unit(s) for project: %s\n", len(units), o.projectName)
	}

	return nil
}

// Stop stops all units for the project, or specific services if provided.
func (o *Orchestrator) Stop(services []string) error {
	units, err := o.resolveUnits(services)
	if err != nil {
		return err
	}

	dbusMgr, err := deploy.NewSystemdManager()
	if err != nil {
		return err
	}
	defer dbusMgr.Close()

	for _, unitName := range units {
		fmt.Printf("Stopping unit: %s\n", unitName)
		if err := dbusMgr.StopUnit(unitName); err != nil {
			return fmt.Errorf("failed to stop unit %s: %w", unitName, err)
		}
	}

	// Verify units are actually stopped (reuse the existing D-Bus connection)
	if err := o.verifyUnitsStoppedByNames(dbusMgr, units); err != nil {
		return err
	}

	if len(services) == 0 {
		fmt.Printf("Successfully stopped project: %s\n", o.projectName)
	} else {
		fmt.Printf("Successfully stopped %d unit(s) for project: %s\n", len(units), o.projectName)
	}

	return nil
}

// Restart restarts all units for the project, or specific services if provided.
func (o *Orchestrator) Restart(services []string) error {
	units, err := o.resolveUnits(services)
	if err != nil {
		return err
	}

	dbusMgr, err := deploy.NewSystemdManager()
	if err != nil {
		return err
	}
	defer dbusMgr.Close()

	for _, unitName := range units {
		fmt.Printf("Restarting unit: %s\n", unitName)
		if err := dbusMgr.RestartUnit(unitName); err != nil {
			return fmt.Errorf("failed to restart unit %s: %w", unitName, err)
		}
	}

	if len(services) == 0 {
		fmt.Printf("Successfully restarted project: %s\n", o.projectName)
	} else {
		fmt.Printf("Successfully restarted %d unit(s) for project: %s\n", len(units), o.projectName)
	}

	return nil
}

// verifyUnitsStoppedByNames verifies that the given unit names are no longer active.
func (o *Orchestrator) verifyUnitsStoppedByNames(dbusMgr *deploy.SystemdManager, unitNames []string) error {
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
