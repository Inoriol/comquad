package preprocess

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// StringMap stores key-value pairs and supports both YAML map and list formats
// during unmarshaling (e.g. `KEY: value` or `- KEY=value`).
// During marshaling it always outputs as a list, which is the standard Docker
// Compose format and what downstream tools like podlet expect.
type StringMap map[string]string

func (sm *StringMap) UnmarshalYAML(node *yaml.Node) error {
	// Try list format first: `- KEY=value`
	var list []string
	if err := node.Decode(&list); err == nil {
		*sm = make(map[string]string)
		for _, item := range list {
			if idx := strings.Index(item, "="); idx >= 0 {
				(*sm)[item[:idx]] = item[idx+1:]
			}
		}
		return nil
	}
	// Fall back to map format: `KEY: value`
	var m map[string]string
	if err := node.Decode(&m); err != nil {
		return err
	}
	*sm = m
	return nil
}

func (sm StringMap) MarshalYAML() (interface{}, error) {
	pairs := make([]string, 0, len(sm))
	for k, v := range sm {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	return pairs, nil
}

// ComposeFile represents the top-level structure of a docker-compose.yaml.
// Services and Volumes use generic maps to preserve all fields (including
// unknown ones like depends_on, restart, etc.) through the unmarshal/marshal cycle.
type ComposeFile struct {
	Services map[string]map[string]interface{} `yaml:"services"`
	Networks map[string]interface{}             `yaml:"networks,omitempty"`
	Volumes  map[string]map[string]interface{} `yaml:"volumes,omitempty"`
	Config   *ProjectConfig                      `yaml:"-"` // Internal use
}

// ProjectConfig holds metadata injected during pre-processing
type ProjectConfig struct {
	ProjectName      string
	WorkingDirectory string
}

// ServiceImageSpec holds the compose fields that map to a Podman .image quadlet.
type ServiceImageSpec struct {
	ServiceName string // compose service name, e.g. "web"
	Image       string // normalized image, e.g. "docker.io/library/nginx:latest"
	PullPolicy  string // pull_policy value: always, missing, never, if_not_present, etc.
	OS          string // from platform field, e.g. "linux"
	Arch        string // from platform field, e.g. "amd64"
	Variant     string // from platform field, e.g. "v8"
}
