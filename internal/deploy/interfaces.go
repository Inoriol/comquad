package deploy

import (
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

// SystemdClient is the subset of SystemdManager used by the orchestrator.
// Using an interface here allows tests to inject a fake without a live D-Bus.
type SystemdClient interface {
	Close() error
	ReloadDaemon(filePaths ...string) error
	WaitForUnit(unitName string, timeout time.Duration) error
	StartUnit(unitName string) error
	StopUnit(unitName string) error
	RestartUnit(unitName string) error
	ListUnitsByNames(unitNames []string) ([]dbus.UnitStatus, error)
	ListAllUnits() ([]dbus.UnitStatus, error)
	GetInvocationID(unitName string) (string, error)
}

// StateStore is the subset of StateManager used by the orchestrator.
// GetProject replaces direct map access so the interface is satisfiable.
type StateStore interface {
	GetProject(name string) (ProjectState, bool)
	GetStateFilePath() string
	ListProjects() []ProjectState
	RegisterProject(project ProjectState) error
	UnregisterProject(projectName string) error
	Save() error
}

// Ensure the concrete types satisfy their interfaces at compile time.
var _ SystemdClient = (*SystemdManager)(nil)
var _ StateStore = (*StateManager)(nil)
