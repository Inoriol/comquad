package orchestrator

import (
	"fmt"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
)

// Ps shows the current state of units for the project.
func (o *Orchestrator) Ps() error {
	stateMgr, err := o.newState()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	if _, exists := stateMgr.GetProject(o.projectName); !exists {
		return fmt.Errorf("project %s is not deployed", o.projectName)
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	units, err := dbusMgr.ListAllUnits()
	if err != nil {
		return fmt.Errorf("failed to list units: %w", err)
	}

	prefix := "cq-" + o.projectName
	var projectUnits []dbus.UnitStatus
	for _, u := range units {
		if strings.HasPrefix(u.Name, prefix) {
			projectUnits = append(projectUnits, u)
		}
	}

	if len(projectUnits) == 0 {
		fmt.Printf("No units found for project %s\n", o.projectName)
		return nil
	}

	fmt.Printf("%-40s %-10s %-10s\n", "UNIT", "ACTIVE", "SUB")
	fmt.Println(strings.Repeat("-", 60))
	for _, u := range projectUnits {
		fmt.Printf("%-40s %-10s %-10s\n", u.Name, u.ActiveState, u.SubState)
	}

	return nil
}
