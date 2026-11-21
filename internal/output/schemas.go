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
	Status     string          `json:"status"`
	Services   string          `json:"services"`
	Jobs       string          `json:"jobs,omitempty"`
	Resources  *ResourcesJSON  `json:"resources,omitempty"`
}

type ResourcesJSON struct {
	Containers []string `json:"containers"`
	Networks   []string `json:"networks"`
	Volumes    []string `json:"volumes"`
	Images     []string `json:"images"`
	Builds     []string `json:"builds"`
	Secrets    []string `json:"secrets,omitempty"`
	Timers     []string `json:"timers,omitempty"`
}

type PSData struct {
	Containers []ContainerJSON `json:"containers"`
}

type ContainerJSON struct {
	Name         string        `json:"name"`
	Image        string        `json:"image"`
	Command      string        `json:"command"`
	Service      string        `json:"service"`
	Type         string        `json:"type"`
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

type ViewData struct {
	Project    string         `json:"project"`
	SourcePath string         `json:"source_path"`
	Status     string         `json:"status"`
	Services   []ServiceJSON  `json:"services"`
	Jobs       []JobJSON      `json:"jobs,omitempty"`
	Resources  []ResourceJSON `json:"resources"`
}

type ServiceJSON struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Image    string   `json:"image"`
	Networks []string `json:"networks"`
	Volumes  []string `json:"volumes"`
}

type JobJSON struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Image     string `json:"image"`
	Schedule  string `json:"schedule,omitempty"`
	Manual    bool   `json:"manual"`
	LastRun   string `json:"last_run,omitempty"`
}

type ResourceJSON struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Info string `json:"info,omitempty"`
}

type UnitFileJSON struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type LifecycleData struct {
	Action  string   `json:"action"`
	Project string   `json:"project"`
	Units   []string `json:"units"`
	Success bool     `json:"success"`
}

type DownData struct {
	Project         string   `json:"project"`
	RemovedFiles    []string `json:"removed_files"`
	RemovedNetworks []string `json:"removed_networks"`
	RemovedVolumes  []string `json:"removed_volumes,omitempty"`
	Success         bool     `json:"success"`
}

type UpData struct {
	Project      string   `json:"project"`
	SourcePath   string   `json:"source_path"`
	FilesWritten []string `json:"files_written"`
	FilesRemoved []string `json:"files_removed,omitempty"`
	UnitsStarted []string `json:"units_started"`
	Success      bool     `json:"success"`
}

type LogsData struct {
	Project string         `json:"project"`
	Entries []LogEntryJSON `json:"entries"`
}

type LogEntryJSON struct {
	Timestamp string `json:"timestamp"`
	Unit      string `json:"unit"`
	Priority  int    `json:"priority"`
	Message   string `json:"message"`
}

type DryRunData struct {
	Project      string           `json:"project"`
	TargetDir    string           `json:"target_dir"`
	PullStrategy string           `json:"pull_strategy"`
	Images       []DryRunImage    `json:"images,omitempty"`
	Builds       []DryRunBuild    `json:"builds,omitempty"`
	Files        []DryRunFile     `json:"files,omitempty"`
	HasChanges   bool             `json:"has_changes"`
}

type DryRunImage struct {
	Name   string `json:"name"`
	Ref    string `json:"ref"`
	Action string `json:"action"`
}

type DryRunBuild struct {
	Name     string `json:"name"`
	ImageTag string `json:"image_tag"`
}

type DryRunFile struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Status     string `json:"status"`
	Diff       string `json:"diff"`
	NewContent string `json:"new_content,omitempty"`
}

type PullData struct {
	Project      string   `json:"project"`
	ImagesPulled []string `json:"images_pulled"`
	Success      bool     `json:"success"`
}

type RunData struct {
	Project string `json:"project"`
	Job     string `json:"job"`
	Action  string `json:"action"`
	DryRun  bool   `json:"dry_run,omitempty"`
	Success bool   `json:"success,omitempty"`
}
