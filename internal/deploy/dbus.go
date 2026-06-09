package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	if isRootless := os.Getuid() != 0; isRootless {
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

// ListUnitsByNames returns the current state of the specified units.
func (s *SystemdManager) ListUnitsByNames(unitNames []string) ([]dbus.UnitStatus, error) {
	return s.conn.ListUnitsByNamesContext(context.Background(), unitNames)
}

// ListAllUnits returns all units known to systemd.
func (s *SystemdManager) ListAllUnits() ([]dbus.UnitStatus, error) {
	return s.conn.ListUnitsContext(context.Background())
}
