package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"

	"comquad/internal/deploy"
	"comquad/internal/logger"
)

// View shows systemd units for a project or the contents of a specific unit file.
func (o *Orchestrator) View(projectArg string) error {
	_, state, err := o.ensureProjectDeployed()
	if err != nil {
		return err
	}

	if projectArg == "" {
		return o.viewProject(state)
	}

	return o.viewUnit(state, projectArg)
}

func (o *Orchestrator) viewProject(state deploy.ProjectState) error {
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
		logger.Printf("No units found for project %s\n", o.projectName)
		return nil
	}

	activeCount := 0
	for _, u := range projectUnits {
		if u.ActiveState == "active" {
			activeCount++
		}
	}

	var status string
	switch {
	case activeCount == len(projectUnits):
		status = "healthy"
	case activeCount == 0:
		status = "down"
	default:
		status = "degraded"
	}

	logger.Printf("%-12s %-40s %s\n", "PROJECT", "SOURCE", "STATUS")
	logger.Print(strings.Repeat("-", 62))
	logger.Printf("%-12s %-40s %s\n", o.projectName, state.SourcePath, status)
	logger.Print("")

	logger.Printf("%-40s %-10s %-10s\n", "UNIT", "ACTIVE", "SUB")
	logger.Print(strings.Repeat("-", 60))
	for _, u := range projectUnits {
		logger.Printf("%-40s %-10s %-10s\n", u.Name, u.ActiveState, u.SubState)
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
	return MatchFirstContainer(o.projectName, state, arg)
}

func (o *Orchestrator) matchNetworkOrVolume(state deploy.ProjectState, arg string) string {
	return MatchNetworkOrVolume(o.projectName, state, arg)
}

func (o *Orchestrator) printFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	logger.Printf("%s", string(content))
	if !strings.HasSuffix(string(content), "\n") {
		logger.Print("")
	}

	return nil
}
