package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"comquad/internal/deploy"
)

// Logs prints logs for a deployed project's services via journalctl.
// For running units, filters logs to the current invocation ID.
// For non-running units, shows full historical logs.
// If follow is true, streams logs continuously.
// If services is empty, logs from all services are shown.
func (o *Orchestrator) Logs(services []string, follow bool) error {
	stateMgr, err := deploy.NewStateManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	state, exists := stateMgr.Projects[o.projectName]
	if !exists {
		return fmt.Errorf("project %s is not deployed", o.projectName)
	}

	// Filter to .container files, optionally by service name
	var unitNames []string
	seen := make(map[string]bool)
	for _, s := range services {
		for _, f := range MatchContainers(o.projectName, state, s) {
			unitName := ContainerFileToUnitName(f)
			if !seen[unitName] {
				seen[unitName] = true
				unitNames = append(unitNames, unitName)
			}
		}
	}
	if len(services) == 0 {
		for _, f := range state.Files {
			if !strings.HasSuffix(f, ".container") {
				continue
			}
			unitName := ContainerFileToUnitName(f)
			if !seen[unitName] {
				seen[unitName] = true
				unitNames = append(unitNames, unitName)
			}
		}
	}

	if len(unitNames) == 0 {
		if len(services) > 0 {
			return fmt.Errorf("no service matching %s found in project %s", strings.Join(services, ", "), o.projectName)
		}
		return fmt.Errorf("no container units found for project %s", o.projectName)
	}

	dbusMgr, err := deploy.NewSystemdManager()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	// Group units by invocation ID (only running units have one)
	invocationGroups := make(map[string][]string)
	var nonRunningUnits []string

	for _, unit := range unitNames {
		status, err := dbusMgr.ListUnitsByNames([]string{unit})
		if err != nil {
			return fmt.Errorf("failed to get status for unit %s: %w", unit, err)
		}
		if len(status) == 0 {
			nonRunningUnits = append(nonRunningUnits, unit)
			continue
		}

		if status[0].ActiveState == "active" {
			invocationID, err := dbusMgr.GetInvocationID(unit)
			if err != nil {
				return fmt.Errorf("failed to get invocation ID for unit %s: %w", unit, err)
			}
			if invocationID != "" {
				invocationGroups[invocationID] = append(invocationGroups[invocationID], unit)
			} else {
				nonRunningUnits = append(nonRunningUnits, unit)
			}
		} else {
			nonRunningUnits = append(nonRunningUnits, unit)
		}
	}

	// Run journalctl for each invocation group
	for invocationID, units := range invocationGroups {
		if err := o.runJournalctl(units, invocationID, follow); err != nil {
			return err
		}
	}

	// Run journalctl for non-running units (full history)
	if len(nonRunningUnits) > 0 {
		if err := o.runJournalctl(nonRunningUnits, "", follow); err != nil {
			return err
		}
	}

	return nil
}

func (o *Orchestrator) runJournalctl(unitNames []string, invocationID string, follow bool) error {
	args := []string{"--no-pager"}
	if os.Getuid() == 0 {
		args = append(args, "--system")
	} else {
		args = append(args, "--user")
	}
	if follow {
		args = append(args, "-f")
	}
	for _, unit := range unitNames {
		args = append(args, "-u", unit)
	}
	if invocationID != "" {
		args = append(args, "--invocation="+invocationID)
	}

	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
