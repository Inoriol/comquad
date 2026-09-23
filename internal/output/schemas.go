package output

import "time"

const APIVersion = "1.0"

type Envelope struct {
	Version string      `json:"version"`
	Data    interface{} `json:"data"`
}

type ErrorEnvelope struct {
	Version string     `json:"version"`
	Error   *ErrorInfo `json:"error"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ProjectListData struct {
	Projects []ProjectJSON `json:"projects"`
}

type ProjectJSON struct {
	Name       string          `json:"name"`
	SourcePath string          `json:"source_path"`
	Files      int             `json:"files"`
	Resources  *ResourcesJSON  `json:"resources,omitempty"`
}

type ResourcesJSON struct {
	Containers []string `json:"containers"`
	Networks   []string `json:"networks"`
	Volumes    []string `json:"volumes"`
	Images     []string `json:"images"`
	Builds     []string `json:"builds"`
}

type PSData struct {
	Containers []ContainerJSON `json:"containers"`
}

type ContainerJSON struct {
	Name         string        `json:"name"`
	Image        string        `json:"image"`
	Command      string        `json:"command"`
	Service      string        `json:"service"`
	State        string        `json:"state"`
	Status       string        `json:"status"`
	Ports        []PortJSON    `json:"ports"`
	ExposedPorts []string      `json:"exposed_ports,omitempty"`
	Networks     []string      `json:"networks,omitempty"`
	Mounts       []string      `json:"mounts,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	ExitedAt     time.Time     `json:"exited_at,omitempty"`
	ExitCode     int           `json:"exit_code,omitempty"`
	DBusActive   string        `json:"dbus_active,omitempty"`
	DBusSub      string        `json:"dbus_sub,omitempty"`
}

type PortJSON struct {
	Protocol      string `json:"protocol"`
	ContainerPort int    `json:"container_port"`
	HostIP        string `json:"host_ip"`
	HostPort      int    `json:"host_port"`
}
