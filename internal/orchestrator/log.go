package orchestrator

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Logs prints logs for a deployed project's services via journalctl.
// For running units, filters logs to the current invocation ID.
// For non-running units, shows full historical logs.
// If follow is true, streams logs continuously.
// If services is empty, logs from all services are shown.
// tail, since, and output are journalctl flags.
func (o *Orchestrator) Logs(services []string, follow bool, tail string, since string, output string) error {
	stateMgr, err := o.newState()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	state, exists := stateMgr.GetProject(o.projectName)
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

	dbusMgr, err := o.newSystemd()
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

	multiUnit := len(unitNames) > 1

	// Run journalctl for each invocation group
	for invocationID, units := range invocationGroups {
		if err := o.runJournalctl(units, invocationID, follow, tail, since, output, multiUnit); err != nil {
			return err
		}
	}

	// Run journalctl for non-running units (full history)
	if len(nonRunningUnits) > 0 {
		if err := o.runJournalctl(nonRunningUnits, "", follow, tail, since, output, multiUnit); err != nil {
			return err
		}
	}

	return nil
}

var execCommand = exec.Command

func writeFilteredLines(cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start journalctl: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			fmt.Println(line)
		}
	}

	return cmd.Wait()
}

func (o *Orchestrator) runJournalctl(unitNames []string, invocationID string, follow bool, tail, since, output string, prefix bool) error {
	args := []string{"--no-pager"}

	// Default output format to short-iso to strip systemd metadata
	if output == "" {
		args = append(args, "--output=cat")
	} else {
		args = append(args, "--output="+output)
	}

	if os.Getuid() == 0 {
		args = append(args, "--system")
	} else {
		args = append(args, "--user")
	}
	if follow {
		args = append(args, "-f")
	}
	if tail != "" {
		args = append(args, "-n", tail)
	}
	if since != "" {
		args = append(args, "--since="+since)
	}
	for _, unit := range unitNames {
		args = append(args, "-u", unit)
	}
	if invocationID != "" {
		args = append(args, "--invocation="+invocationID)
	}

	if prefix {
		return o.runJournalctlWithPrefix(unitNames, args, follow)
	}

	cmd := execCommand("journalctl", args...)
	cmd.Stderr = os.Stderr

	return writeFilteredLines(cmd)
}

func (o *Orchestrator) runJournalctlWithPrefix(unitNames []string, args []string, follow bool) error {
	// Run journalctl without -u flags (we'll filter by unit ourselves)
	filteredArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-u ") {
			continue
		}
		if strings.HasPrefix(arg, "-u=") {
			continue
		}
		filteredArgs = append(filteredArgs, arg)
	}

	cmd := execCommand("journalctl", filteredArgs...)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start journalctl: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if len(unitNames) == 0 {
			fmt.Println(line)
			continue
		}
		prefix := "[" + unitNames[0] + "] "
		if !strings.HasPrefix(line, prefix) && !strings.HasPrefix(line, "--") {
			fmt.Println(prefix + line)
		}
	}

	return cmd.Wait()
}

// FollowLogs streams all journalctl logs for every unit in the project
// from the given timestamp onward.
func (o *Orchestrator) FollowLogs(since string, tail, output string) error {
	stateMgr, err := o.newState()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	state, exists := stateMgr.GetProject(o.projectName)
	if !exists {
		return fmt.Errorf("project %s is not deployed", o.projectName)
	}

	var unitNames []string
	for _, f := range state.Files {
		var unitName string
		switch {
		case strings.HasSuffix(f, ".container"):
			unitName = ContainerFileToUnitName(f)
		case strings.HasSuffix(f, ".network"):
			unitName = NetworkFileToUnitName(f)
		case strings.HasSuffix(f, ".volume"):
			unitName = VolumeFileToUnitName(f)
		}
		if unitName != "" {
			unitNames = append(unitNames, unitName)
		}
	}

	if len(unitNames) == 0 {
		return fmt.Errorf("no units found for project %s", o.projectName)
	}

	args := []string{"--no-pager", "--since=" + since, "-f"}

	// Default output format to short-iso to strip systemd metadata
	if output == "" {
		args = append(args, "--output=cat")
	} else {
		args = append(args, "--output="+output)
	}

	if os.Getuid() == 0 {
		args = append(args, "--system")
	} else {
		args = append(args, "--user")
	}
	if tail != "" {
		args = append(args, "-n", tail)
	}
	for _, unit := range unitNames {
		args = append(args, "-u", unit)
	}

	cmd := execCommand("journalctl", args...)
	cmd.Stderr = os.Stderr

	return writeFilteredLines(cmd)
}
