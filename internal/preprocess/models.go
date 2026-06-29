package preprocess

import (
	"fmt"
	"sort"
	"strconv"
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

// BuildConfig holds parsed build configuration from a compose service
type BuildConfig struct {
	Context    string            `yaml:"context,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty"`
	Args       map[string]string `yaml:"args,omitempty"`
	Target     string            `yaml:"target,omitempty"`
}

// BuildInfo holds the build information for a service after preprocessing
type BuildInfo struct {
	Context    string
	Dockerfile string
	Args       []string
	Target     string
	Service    string
}

// ProjectConfig holds metadata injected during pre-processing
type ProjectConfig struct {
	ProjectName      string
	WorkingDirectory string
}

// buildArgValue converts a YAML build arg value to its string representation.
// Handles string, bool, integer, float, and nil types explicitly to avoid
// surprises from fmt.Sprintf("%v", val) (e.g. nil becoming "<nil>").
func buildArgValue(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		// YAML unmarshals all numbers as float64; format without trailing zeros
		// for integers (e.g. 1.0 -> "1", 1.5 -> "1.5")
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}
