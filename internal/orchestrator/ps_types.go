package orchestrator

import "time"

// ContainerInfo holds parsed podman ps JSON data for a single container.
type ContainerInfo struct {
	Name         string     `json:"name"`
	Image        string     `json:"image"`
	Command      string     `json:"command"`
	Service      string     `json:"service"`
	State        string     `json:"state"`
	Status       string     `json:"status"`
	Ports        []PortInfo `json:"ports"`
	ExposedPorts []string   `json:"exposed_ports,omitempty"`
	Networks     []string   `json:"networks,omitempty"`
	Mounts       []string   `json:"mounts,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ExitedAt     time.Time  `json:"exited_at,omitempty"`
	ExitCode     int        `json:"exit_code,omitempty"`
	DBusActive   string     `json:"dbus_active,omitempty"`
	DBusSub      string     `json:"dbus_sub,omitempty"`
}

// PortInfo holds published port mapping for a container.
type PortInfo struct {
	Protocol      string `json:"protocol"`
	ContainerPort int    `json:"container_port"`
	HostIP        string `json:"host_ip"`
	HostPort      int    `json:"host_port"`
}
