package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"comquad/internal/deploy"
)

// Logs prints logs for a deployed project's services via journalctl.
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
	for _, f := range state.Files {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		unitName := ContainerFileToUnitName(f)
		if len(services) == 0 {
			unitNames = append(unitNames, unitName)
		} else {
			for _, s := range services {
				if strings.Contains(unitName, s) {
					unitNames = append(unitNames, unitName)
					break
				}
			}
		}
	}

	if len(unitNames) == 0 {
		if len(services) > 0 {
			return fmt.Errorf("no service matching %s found in project %s", strings.Join(services, ", "), o.projectName)
		}
		return fmt.Errorf("no container units found for project %s", o.projectName)
	}

	// Build journalctl command with -u flags for each unit
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

	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
