package orchestrator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// journalEntry represents a parsed journalctl JSON log entry.
type journalEntry struct {
	timestamp int64
	unit      string
	message   string
	priority  int
}

// priorityText maps syslog priority to human-readable text.
func priorityText(prio int) string {
	switch prio {
	case 0:
		return "EMERG"
	case 1:
		return "ALERT"
	case 2:
		return "CRIT"
	case 3:
		return "ERR"
	case 4:
		return "WARNING"
	case 5:
		return "NOTICE"
	case 6:
		return "INFO"
	case 7:
		return "DEBUG"
	default:
		return fmt.Sprintf("P%d", prio)
	}
}

// parseJournalEntry extracts fields from a journalctl JSON line.
func parseJournalEntry(line string) (journalEntry, bool) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return journalEntry{}, false
	}

	entry := journalEntry{}

	if tsRaw, ok := raw["__REALTIME_TIMESTAMP"]; ok {
		switch v := tsRaw.(type) {
		case string:
			if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
				entry.timestamp = ts
			}
		case float64:
			entry.timestamp = int64(v)
		}
	}

	if unit, ok := raw["SYSTEMD_UNIT"]; ok {
		entry.unit = unit.(string)
	} else if unit, ok := raw["_SYSTEMD_USER_UNIT"]; ok {
		entry.unit = unit.(string)
	}

	if msg, ok := raw["MESSAGE"]; ok {
		if s, ok := msg.(string); ok {
			entry.message = strings.ReplaceAll(s, "\n", " ")
		}
	}

	if prioRaw, ok := raw["PRIORITY"]; ok {
		switch v := prioRaw.(type) {
		case float64:
			entry.priority = int(v)
		case string:
			if p, err := strconv.Atoi(v); err == nil {
				entry.priority = p
			}
		}
	}

	return entry, true
}

// renderEntry formats a journal entry for output.
func renderEntry(entry journalEntry, showTime bool) string {
	var parts []string
	if showTime {
		sec := entry.timestamp / 1e6
		nsec := (entry.timestamp % 1e6) * 1e6
		t := time.Unix(sec, nsec).UTC().Format(time.RFC3339Nano)
		parts = append(parts, t)
	}
	unitStr := entry.unit
	if unitStr == "" {
		unitStr = "?"
	}
	parts = append(parts, "["+unitStr+"]")
	parts = append(parts, priorityText(entry.priority)+": "+entry.message)
	return strings.Join(parts, "  ")
}

// flushEntries sorts entries by timestamp and renders them.
func flushEntries(entries []journalEntry, showTime bool) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].timestamp < entries[j].timestamp
	})
	for _, e := range entries {
		fmt.Println(renderEntry(e, showTime))
	}
}

// Logs prints logs for a deployed project's services via journalctl.
func (o *Orchestrator) Logs(services []string, follow bool, tail, since string, showTime bool) error {
	_, state, err := o.ensureProjectDeployed()
	if err != nil {
		return err
	}

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

	// --- Follow mode: process each group separately (can't sort live stream) ---
	if follow {
		for invocationID, units := range invocationGroups {
			if err := o.runJournalctlJSONFollowForGroup(units, invocationID, tail, since, showTime); err != nil {
				return err
			}
		}
		if len(nonRunningUnits) > 0 {
			if err := o.runJournalctlJSONFollowForGroup(nonRunningUnits, "", tail, since, showTime); err != nil {
				return err
			}
		}
		return nil
	}

	// --- Batch mode: collect ALL entries, sort together, render once ---
	var allEntries []journalEntry

	for invocationID, units := range invocationGroups {
		entries, err := o.collectJournalEntries(units, invocationID, tail, since)
		if err != nil {
			return err
		}
		allEntries = append(allEntries, entries...)
	}

	if len(nonRunningUnits) > 0 {
		entries, err := o.collectJournalEntries(nonRunningUnits, "", tail, since)
		if err != nil {
			return err
		}
		allEntries = append(allEntries, entries...)
	}

	flushEntries(allEntries, showTime)
	return nil
}

var execCommand = exec.Command

// collectJournalEntries runs journalctl and returns parsed entries (no sorting).
func (o *Orchestrator) collectJournalEntries(unitNames []string, invocationID, tail, since string) ([]journalEntry, error) {
	args := []string{"--no-pager", "--output=json"}

	if os.Getuid() == 0 {
		args = append(args, "--system")
	} else {
		args = append(args, "--user")
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

	cmd := execCommand("journalctl", args...)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start journalctl: %w", err)
	}

	var entries []journalEntry
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if entry, ok := parseJournalEntry(line); ok {
			entries = append(entries, entry)
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("journalctl failed: %w", err)
	}

	return entries, nil
}

// runJournalctlJSONFollowForGroup runs journalctl for a single group in follow mode.
func (o *Orchestrator) runJournalctlJSONFollowForGroup(unitNames []string, invocationID, tail, since string, showTime bool) error {
	args := []string{"--no-pager", "--since=" + since, "-f", "--output=json"}

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
	if invocationID != "" {
		args = append(args, "--invocation="+invocationID)
	}

	cmd := execCommand("journalctl", args...)
	cmd.Stderr = os.Stderr

	return o.runJournalctlJSONFollow(cmd, showTime)
}

// runJournalctlJSONFollow streams JSON output, buffers entries, and flushes them in timestamp order.
func (o *Orchestrator) runJournalctlJSONFollow(cmd *exec.Cmd, showTime bool) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start journalctl: %w", err)
	}

	var mu sync.Mutex
	var entries []journalEntry
	done := make(chan struct{})

	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			if entry, ok := parseJournalEntry(line); ok {
				mu.Lock()
				entries = append(entries, entry)
				mu.Unlock()
			}
		}
		close(done)
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			mu.Lock()
			flushEntries(entries, showTime)
			mu.Unlock()
			return cmd.Wait()
		case <-ticker.C:
			mu.Lock()
			if len(entries) > 0 {
				flushEntries(entries, showTime)
				entries = nil
			}
			mu.Unlock()
		}
	}
}

// FollowLogs streams all journalctl logs for every unit in the project
// from the given timestamp onward.
func (o *Orchestrator) FollowLogs(since, tail string, showTime bool) error {
	_, state, err := o.ensureProjectDeployed()
	if err != nil {
		return err
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

	args := []string{"--no-pager", "--since=" + since, "-f", "--output=json"}

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

	return o.runJournalctlJSONFollow(cmd, showTime)
}
