package preprocess

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents the top-level structure of a docker-compose.yaml
type ComposeFile struct {
	Services map[string]*Service `yaml:"services"`
	Networks map[string]*Network `yaml:"networks,omitempty"`
	Volumes  map[string]*Volume  `yaml:"volumes,omitempty"`
	Config   *ProjectConfig      `yaml:"-"` // Internal use
}

// serviceBase holds fields shared between Service and serviceWithRaw to avoid
// duplicating the yaml struct tags in both types.
type serviceBase struct {
	ContainerName string            `yaml:"container_name,omitempty"`
	Image         string            `yaml:"image,omitempty"`
	Ports         []string          `yaml:"ports,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	Networks      []string          `yaml:"networks,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	Deploy        interface{}       `yaml:"deploy,omitempty"`
}

// serviceWithRaw embeds serviceBase and adds the raw build field so the YAML
// decoder can handle build as either a string or a map.
type serviceWithRaw struct {
	serviceBase `yaml:",inline"`
	BuildRaw    interface{} `yaml:"build,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling to handle build field
func (s *Service) UnmarshalYAML(node *yaml.Node) error {
	var raw serviceWithRaw
	if err := node.Decode(&raw); err != nil {
		return err
	}

	s.ContainerName = raw.ContainerName
	s.Image = raw.Image
	s.Ports = raw.Ports
	s.Volumes = raw.Volumes
	s.Networks = raw.Networks
	s.Environment = raw.Environment
	s.Labels = raw.Labels
	s.Deploy = raw.Deploy

	// Handle build field - can be string or map
	if raw.BuildRaw != nil {
		switch v := raw.BuildRaw.(type) {
		case string:
			s.Build = &BuildConfig{
				Context:    v,
				Dockerfile: "Dockerfile",
			}
		case map[string]interface{}:
			config := &BuildConfig{}
			if ctx, ok := v["context"].(string); ok {
				config.Context = ctx
			}
			if df, ok := v["dockerfile"].(string); ok {
				config.Dockerfile = df
			}
			if target, ok := v["target"].(string); ok {
				config.Target = target
			}
			if args, ok := v["args"].(map[string]interface{}); ok {
				config.Args = make(map[string]string)
				for k, val := range args {
					config.Args[k] = buildArgValue(val)
				}
			}
			s.Build = config
		}
	}

	return nil
}

// Service represents a single container service in the compose file
type Service struct {
	ContainerName string            `yaml:"container_name,omitempty"`
	Image         string            `yaml:"image,omitempty"`
	Build         *BuildConfig      `yaml:"build,omitempty"`
	Ports         []string          `yaml:"ports,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	Networks      []string          `yaml:"networks,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	Deploy        interface{}       `yaml:"deploy,omitempty"`
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

// Network represents a network definition
type Network struct {
	Driver string            `yaml:"driver,omitempty"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

// Volume represents a volume definition
type Volume struct {
	Driver string `yaml:"driver,omitempty"`
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
		// for integers (e.g. 1.0 → "1", 1.5 → "1.5")
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}
