package deploy

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"comquad/internal/logger"
	"github.com/coreos/go-systemd/v22/dbus"
)

// SystemdManager handles direct communication with the systemd D-Bus
type SystemdManager struct {
	conn *dbus.Conn
}

// NewSystemdManager initializes a new connection to the systemd bus.
// Uses the user bus for rootless (non-root) and system bus for root.
func NewSystemdManager() (*SystemdManager, error) {
	var conn *dbus.Conn
	var err error

	// Use Background context — connection is long-lived,
	// timeout contexts are only for individual operations
	if IsRootless() {
		conn, err = dbus.NewUserConnectionContext(context.Background())
	} else {
		conn, err = dbus.NewSystemConnectionContext(context.Background())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd bus: %w", err)
	}

	return &SystemdManager{conn: conn}, nil
}

// Close closes the D-Bus connection
func (s *SystemdManager) Close() error {
	s.conn.Close()
	return nil
}

// ReloadDaemon triggers a systemd daemon-reload and waits for quadlet
// generators to produce units for the given file paths.
func (s *SystemdManager) ReloadDaemon(filePaths ...string) error {
	if err := s.conn.Reload(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	if len(filePaths) == 0 {
		return nil
	}

	// Wait for quadlet-generated units to appear
	for _, f := range filePaths {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		unitName := strings.TrimSuffix(filepath.Base(f), ".container") + ".service"
		if err := s.WaitForUnit(unitName, 15*time.Second); err != nil {
			return fmt.Errorf("quadlet generator did not produce unit %s after reload: %w", unitName, err)
		}
	}

	return nil
}

// WaitForUnit polls systemd until the unit is known or timeout is reached.
// This is needed because the quadlet generator runs asynchronously after daemon-reload.
func (s *SystemdManager) WaitForUnit(unitName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		units, err := s.conn.ListUnitsByNamesContext(context.Background(), []string{unitName})
		if err != nil {
			return fmt.Errorf("failed to list units: %w", err)
		}

		// Unit is known to systemd when it has a load state other than "not-found"
		if len(units) > 0 && units[0].LoadState != "not-found" {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for unit %s to appear in systemd", unitName)
}

// StartUnit starts a specific systemd unit and waits for the job to complete.
// Returns an error if the unit fails to start or the job does not complete with "done".
func (s *SystemdManager) StartUnit(unitName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ch := make(chan string, 1)
	_, err := s.conn.StartUnitContext(ctx, unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to enqueue start job for unit %s: %w", unitName, err)
	}

	// Wait for the job result or context timeout.
	// Possible results: "done", "failed", "cancelled", "timeout", "dependency", "skipped"
	select {
	case result := <-ch:
		if result != "done" {
			return fmt.Errorf("unit %s failed to start, job result: %s", unitName, result)
		}
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for unit %s to start", unitName)
	}

	return nil
}

// StopUnit stops a specific systemd unit and waits for the job to complete.
// Returns an error if the unit fails to stop or the job does not complete with "done".
func (s *SystemdManager) StopUnit(unitName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ch := make(chan string, 1)
	_, err := s.conn.StopUnitContext(ctx, unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to enqueue stop job for unit %s: %w", unitName, err)
	}

	// Wait for the job result or context timeout.
	select {
	case result := <-ch:
		if result != "done" {
			return fmt.Errorf("unit %s failed to stop, job result: %s", unitName, result)
		}
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for unit %s to stop", unitName)
	}

	return nil
}

// RestartUnit restarts a specific systemd unit and waits for the job to complete.
// Unlike StartUnit, this always tears down and recreates the unit even if already active.
func (s *SystemdManager) RestartUnit(unitName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ch := make(chan string, 1)
	_, err := s.conn.RestartUnitContext(ctx, unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to enqueue restart job for unit %s: %w", unitName, err)
	}

	// Wait for the job result or context timeout.
	// Possible results: "done", "failed", "cancelled", "timeout", "dependency", "skipped"
	select {
	case result := <-ch:
		if result != "done" {
			return fmt.Errorf("unit %s failed to restart, job result: %s", unitName, result)
		}
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for unit %s to restart", unitName)
	}

	return nil
}

// ListUnitsByNames returns the current state of the specified units.
func (s *SystemdManager) ListUnitsByNames(unitNames []string) ([]dbus.UnitStatus, error) {
	return s.conn.ListUnitsByNamesContext(context.Background(), unitNames)
}

// ListAllUnits returns all units known to systemd.
func (s *SystemdManager) ListAllUnits() ([]dbus.UnitStatus, error) {
	return s.conn.ListUnitsContext(context.Background())
}

// GetInvocationID returns the invocation ID of a specific unit as raw hex.
// Returns empty string if the unit is not found or has no invocation ID.
func (s *SystemdManager) GetInvocationID(unitName string) (string, error) {
	props, err := s.conn.GetUnitPropertiesContext(context.Background(), unitName)
	if err != nil {
		return "", fmt.Errorf("failed to get properties for unit %s: %w", unitName, err)
	}
	inv, ok := props["InvocationID"]
	if !ok {
		return "", nil
	}
	switch v := inv.(type) {
	case []byte:
		return hex.EncodeToString(v), nil
	case string:
		return v, nil
	}
	return "", nil
}

// RemoveNetworks removes all Podman networks matching the cq-<projectName> prefix.
func RemoveNetworks(projectName string) error {
	cmd := exec.Command("podman", "network", "ls", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	prefix := "cq-" + projectName + "-"
	var networks []string
	for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, prefix) || strings.HasSuffix(name, "-"+projectName) {
			networks = append(networks, name)
		}
	}

	if len(networks) == 0 {
		return nil
	}

	for _, name := range networks {
		logger.Warn("Removing network: " + name)
		rmCmd := exec.Command("podman", "network", "rm", name)
		if err := rmCmd.Run(); err != nil {
			logger.Warn("Failed to remove network " + name + ": " + err.Error())
		}
	}

	return nil
}

// RemoveVolumes removes all Podman volumes matching the cq-<projectName> prefix.
func RemoveVolumes(projectName string) error {
	cmd := exec.Command("podman", "volume", "ls", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	prefix := "cq-" + projectName + "-"
	var volumes []string
	for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, prefix) || strings.HasSuffix(name, "-"+projectName) {
			volumes = append(volumes, name)
		}
	}

	if len(volumes) == 0 {
		return nil
	}

	for _, name := range volumes {
		logger.Warn("Removing volume: " + name)
		rmCmd := exec.Command("podman", "volume", "rm", name)
		if err := rmCmd.Run(); err != nil {
			logger.Warn("Failed to remove volume " + name + ": " + err.Error())
		}
	}

	return nil
}

// PodmanResource represents a discovered Podman resource with its project label
type PodmanResource struct {
	Name        string
	ProjectName string
	Type        string // "container", "network", "volume"
}

// RegenerateState scans Podman for managed resources and reconstructs the state file.
func RegenerateState() (*StateManager, error) {
	stateMgr, err := NewStateManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Discover containers, networks, and volumes by label
	containers := discoverResources("container", "com.comquad.managed")
	networks := discoverResources("network", "com.comquad.managed")
	volumes := discoverResources("volume", "com.comquad.managed")

	// Group by project
	projects := make(map[string]*ResourceInfo)
	allResources := []PodmanResource{}

	for _, r := range containers {
		allResources = append(allResources, r)
		if _, ok := projects[r.ProjectName]; !ok {
			projects[r.ProjectName] = &ResourceInfo{}
		}
		projects[r.ProjectName].Containers = append(projects[r.ProjectName].Containers, r.Name)
	}

	for _, r := range networks {
		allResources = append(allResources, r)
		if _, ok := projects[r.ProjectName]; !ok {
			projects[r.ProjectName] = &ResourceInfo{}
		}
		projects[r.ProjectName].Networks = append(projects[r.ProjectName].Networks, r.Name)
	}

	for _, r := range volumes {
		allResources = append(allResources, r)
		if _, ok := projects[r.ProjectName]; !ok {
			projects[r.ProjectName] = &ResourceInfo{}
		}
		projects[r.ProjectName].Volumes = append(projects[r.ProjectName].Volumes, r.Name)
	}

	// Resolve quadlet files for each project
	resolver := NewTargetDirResolver()
	targetDir, err := resolver.GetSystemdPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve systemd path: %w", err)
	}

	for projectName, resources := range projects {
		var files []string
		entries, err := os.ReadDir(targetDir)
		if err != nil {
			fmt.Printf("Warning: failed to read target directory %s: %v\n", targetDir, err)
			continue
		}

		prefix := "cq-" + projectName
		for _, f := range entries {
			if strings.HasPrefix(f.Name(), prefix) && (strings.HasSuffix(f.Name(), ".container") ||
				strings.HasSuffix(f.Name(), ".network") || strings.HasSuffix(f.Name(), ".volume")) {
				files = append(files, filepath.Join(targetDir, f.Name()))
			}
		}

		stateMgr.Projects[projectName] = ProjectState{
			ProjectName: projectName,
			SourcePath:  "",
			Files:       files,
			Resources:   resources,
		}
	}

	return stateMgr, nil
}

// discoverResources queries Podman for resources of a given type with the managed label
func discoverResources(resourceType, label string) []PodmanResource {
	var results []PodmanResource

	var cmd *exec.Cmd
	switch resourceType {
	case "container":
		cmd = exec.Command("podman", "ps", "-a", "--filter", "label="+label, "--format", "{{.Names}}|{{.Label \"com.comquad.project\"}}")
	case "network":
		cmd = exec.Command("podman", "network", "ls", "--filter", "label="+label, "--format", "{{.Name}}|{{.Label \"com.comquad.project\"}}")
	case "volume":
		cmd = exec.Command("podman", "volume", "ls", "--filter", "label="+label, "--format", "{{.Name}}|{{.Label \"com.comquad.project\"}}")
	default:
		return results
	}

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Warning: failed to list %ss: %v\n", resourceType, err)
		return results
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		project := parts[1]
		if name == "" || project == "" {
			continue
		}
		results = append(results, PodmanResource{
			Name:        name,
			ProjectName: project,
			Type:        resourceType,
		})
	}

	return results
}
