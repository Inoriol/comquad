package orchestrator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Inoriol/comquad/internal/logger"
)

const (
	logFlushInterval = 500 * time.Millisecond
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
		unitStr = "?" // fallback for entries without SYSTEMD_UNIT or _SYSTEMD_USER_UNIT
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
	var buf strings.Builder
	for _, e := range entries {
		buf.WriteString(renderEntry(e, showTime))
		buf.WriteByte('\n')
	}
	logger.Printf("%s", buf.String())
}

// Logs prints logs for a deployed project's services via journalctl.
func (o *Orchestrator) Logs(services []string, follow bool, tail, since string, showTime bool) error {
	if since != "" {
		normalized, err := normalizeSince(since)
		if err != nil {
			return err
		}
		since = normalized
	}

	_, state, err := o.ensureProjectDeployed()
	if err != nil {
		return err
	}

	var unitNames []string
	seen := make(map[string]bool)
	for _, s := range services {
		for _, f := range MatchAllContainers(o.projectName, state, s) {
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

	// --- Follow mode: run every group concurrently, merging the streams ---
	if follow {
		var cmds []*exec.Cmd
		for invocationID, units := range invocationGroups {
			cmds = append(cmds, o.buildJournalctlFollowCmd(units, invocationID, tail, since))
		}
		if len(nonRunningUnits) > 0 {
			cmds = append(cmds, o.buildJournalctlFollowCmd(nonRunningUnits, "", tail, since))
		}
		return o.runJournalctlFollow(cmds, showTime)
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

	cmd := o.newJournalCmd("journalctl", args...)
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

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed reading journalctl output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("journalctl failed: %w", err)
	}

	return entries, nil
}

// buildJournalctlFollowCmd constructs a journalctl -f command for the given
// units and optional invocation ID.
func (o *Orchestrator) buildJournalctlFollowCmd(unitNames []string, invocationID, tail, since string) *exec.Cmd {
	args := []string{"--no-pager", "-f", "--output=json"}

	if since != "" {
		args = append(args, "--since="+since)
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
	if invocationID != "" {
		args = append(args, "--invocation="+invocationID)
	}

	return o.newJournalCmd("journalctl", args...)
}

// runJournalctlFollow starts every journalctl -f command concurrently, buffers
// their parsed entries, and flushes them in timestamp order every
// logFlushInterval. It returns once all commands have exited (e.g. Ctrl+C).
func (o *Orchestrator) runJournalctlFollow(cmds []*exec.Cmd, showTime bool) error {
	if len(cmds) == 0 {
		return nil
	}

	var mu sync.Mutex
	var entries []journalEntry
	errCh := make(chan error, len(cmds))

	for _, cmd := range cmds {
		cmd.Stderr = os.Stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start journalctl: %w", err)
		}

		go func(cmd *exec.Cmd, stdout io.Reader) {
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
			errCh <- cmd.Wait()
		}(cmd, stdout)
	}

	ticker := time.NewTicker(logFlushInterval)
	defer ticker.Stop()

	var firstErr error
	remaining := len(cmds)
	for remaining > 0 {
		select {
		case err := <-errCh:
			remaining--
			if err != nil && firstErr == nil {
				firstErr = err
			}
		case <-ticker.C:
			mu.Lock()
			if len(entries) > 0 {
				flushEntries(entries, showTime)
				entries = nil
			}
			mu.Unlock()
		}
	}

	mu.Lock()
	flushEntries(entries, showTime)
	mu.Unlock()

	if firstErr != nil {
		return fmt.Errorf("journalctl failed: %w", firstErr)
	}
	return nil
}

// FollowLogs streams all journalctl logs for every unit in the project
// from the given timestamp onward.
func (o *Orchestrator) FollowLogs(since, tail string, showTime bool) error {
	if since != "" {
		normalized, err := normalizeSince(since)
		if err != nil {
			return err
		}
		since = normalized
	}

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
		case strings.HasSuffix(f, ".image"):
			unitName = ImageFileToUnitName(f)
		case strings.HasSuffix(f, ".build"):
			unitName = BuildFileToUnitName(f)
		}
		if unitName != "" {
			unitNames = append(unitNames, unitName)
		}
	}

	if len(unitNames) == 0 {
		return fmt.Errorf("no units found for project %s", o.projectName)
	}

	cmd := o.buildJournalctlFollowCmd(unitNames, "", tail, since)
	return o.runJournalctlFollow([]*exec.Cmd{cmd}, showTime)
}

// normalizeSince validates the --since argument and converts bare durations
// such as "10m" to journalctl's relative-time form "-10m".
func normalizeSince(since string) (string, error) {
	if err := validateSince(since); err != nil {
		return "", err
	}
	if !strings.HasPrefix(since, "-") {
		if _, err := time.ParseDuration(since); err == nil {
			return "-" + since, nil
		}
	}
	return since, nil
}

// validateSince checks that the --since argument is plausibly valid.
// journalctl accepts dates (YYYY-MM-DD), relative times (-10m, 1h ago), and keywords (today, yesterday, now).
func validateSince(since string) error {
	knownWords := map[string]bool{
		"today": true, "yesterday": true, "now": true,
		"boot": true, "reboot": true,
	}
	if knownWords[strings.ToLower(since)] {
		return nil
	}
	if strings.HasPrefix(since, "-") || strings.HasSuffix(since, " ago") {
		return nil
	}
	for _, f := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if _, err := time.Parse(f, since); err == nil {
			return nil
		}
	}
	if duration, err := time.ParseDuration(since); err == nil && duration > 0 {
		return nil
	}
	return fmt.Errorf("invalid --since %q: expected a date (YYYY-MM-DD [HH:MM[:SS]]), relative time (-10m, -1h, 1h ago), or keyword (today, yesterday, now, boot)", since)
}
