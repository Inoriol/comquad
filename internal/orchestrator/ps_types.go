package orchestrator

import "time"

// ContainerInfo holds parsed podman ps JSON data for a single container.
type ContainerInfo struct {
	Name         string
	Image        string
	Command      string
	Service      string
	State        string
	Status       string
	Ports        []PortInfo
	ExposedPorts []string
	Networks     []string
	Mounts       []string
	CreatedAt    time.Time
	ExitedAt     time.Time
	ExitCode     int
	DBusActive   string
	DBusSub      string
}

// PortInfo holds published port mapping for a container.
type PortInfo struct {
	Protocol     string
	ContainerPort int
	HostIP       string
	HostPort     int
}
