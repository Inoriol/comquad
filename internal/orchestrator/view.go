package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"

	"comquad/internal/deploy"
)

// View shows systemd units for a project or the contents of a specific unit file.
func (o *Orchestrator) View(projectArg string) error {
	stateMgr, err := deploy.NewStateManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	state, exists := stateMgr.Projects[o.projectName]
	if !exists {
		return fmt.Errorf("project %s is not deployed", o.projectName)
	}

	if projectArg == "" {
		return o.viewProject(state)
	}

	return o.viewUnit(state, projectArg)
}

func (o *Orchestrator) viewProject(state deploy.ProjectState) error {
	dbusMgr, err := deploy.NewSystemdManager()
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

	healthy := true
	for _, u := range projectUnits {
		if u.ActiveState != "active" {
			healthy = false
			break
		}
	}

	status := "down"
	if healthy {
		status = "healthy"
	} else {
		status = "degraded"
	}

	fmt.Printf("%-12s %-40s %s\n", "PROJECT", "SOURCE", "STATUS")
	fmt.Println(strings.Repeat("-", 62))
	fmt.Printf("%-12s %-40s %s\n", o.projectName, state.SourcePath, status)
	fmt.Println()

	fmt.Printf("%-40s %-10s %-10s\n", "UNIT", "ACTIVE", "SUB")
	fmt.Println(strings.Repeat("-", 60))
	for _, u := range projectUnits {
		fmt.Printf("%-40s %-10s %-10s\n", u.Name, u.ActiveState, u.SubState)
	}

	return nil
}

func (o *Orchestrator) viewUnit(state deploy.ProjectState, arg string) error {
	// Try container files first
	if found := o.matchContainer(state, arg); found != "" {
		return o.printFile(found)
	}

	// Try network/volume files
	if found := o.matchNetworkOrVolume(state, arg); found != "" {
		return o.printFile(found)
	}

	// Not found
	var suggestions []string
	for _, f := range state.Files {
		suggestions = append(suggestions, filepath.Base(f))
	}
	return fmt.Errorf("no unit found for '%s', did you mean: %s", arg, strings.Join(suggestions, ", "))
}

func (o *Orchestrator) matchContainer(state deploy.ProjectState, arg string) string {
	servicePrefix := "cq-" + o.projectName + "-"

	for _, f := range state.Files {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		base := filepath.Base(f)
		nameWithoutExt := strings.TrimSuffix(base, ".container")

		if base == arg {
			return f
		}
		if nameWithoutExt == arg {
			return f
		}
		if strings.TrimSuffix(arg, ".service") == nameWithoutExt {
			return f
		}
		if strings.TrimPrefix(nameWithoutExt, servicePrefix) == arg {
			return f
		}
	}
	return ""
}

func (o *Orchestrator) matchNetworkOrVolume(state deploy.ProjectState, arg string) string {
	for _, f := range state.Files {
		if !strings.HasSuffix(f, ".network") && !strings.HasSuffix(f, ".volume") {
			continue
		}
		base := filepath.Base(f)
		nameWithoutExt := strings.TrimSuffix(base, ".network")
		nameWithoutExt = strings.TrimSuffix(nameWithoutExt, ".volume")

		// Compute the service name (systemd unit name without .service)
		serviceName := nameWithoutExt
		if strings.HasSuffix(base, ".network") {
			serviceName = nameWithoutExt + "-network"
		} else if strings.HasSuffix(base, ".volume") {
			serviceName = nameWithoutExt + "-volume"
		}

		if base == arg {
			return f
		}
		if serviceName == arg {
			return f
		}
		if strings.TrimSuffix(arg, ".service") == serviceName {
			return f
		}
	}
	return ""
}

func (o *Orchestrator) printFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	fmt.Print(string(content))
	if !strings.HasSuffix(string(content), "\n") {
		fmt.Println()
	}

	return nil
}
