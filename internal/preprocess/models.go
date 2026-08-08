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
	Services map[string]map[string]interface{}  `yaml:"services"`
	Networks map[string]interface{}             `yaml:"networks,omitempty"`
	Volumes  map[string]map[string]interface{}  `yaml:"volumes,omitempty"`
	Secrets  map[string]map[string]interface{}  `yaml:"secrets,omitempty"`
	Config   *ProjectConfig                     `yaml:"-"` // Internal use
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

// SecretDef holds a parsed top-level compose secret definition.
type SecretDef struct {
	Name         string // compose key name, e.g. "db_password"
	File         string // absolute path for "file:" secrets
	Environment  string // env var name for "environment:" secrets
	Content      string // resolved value (from env var or .env file), for "environment:" secrets
	External     bool   // true if "external: true"
	ExternalName string // "name:" field for external secrets with alternate lookup
}

// SecretRef describes a service's reference to a top-level secret.
type SecretRef struct {
	Source string // secret name from the top-level secrets section
	Target string // optional custom mount path (defaults to /run/secrets/<source>)
}

// ServiceSecretRefs maps a service name to the list of secrets it references.
type ServiceSecretRefs map[string][]SecretRef
