package preprocess

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents the top-level structure of a docker-compose.yaml
type ComposeFile struct {
	Services map[string]*Service `yaml:"services"`
	Networks map[string]*Network `yaml:"networks,omitempty"`
	Volumes  map[string]*Volume  `yaml:"volumes,omitempty"`
	Config   *ProjectConfig      `yaml:"-"` // Internal use
}

// ServiceWithRaw wraps Service to capture raw build values
type ServiceWithRaw struct {
	ContainerName string            `yaml:"container_name,omitempty"`
	Image         string            `yaml:"image,omitempty"`
	BuildRaw      interface{}       `yaml:"build,omitempty"`
	Ports         []string          `yaml:"ports,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	Networks      []string          `yaml:"networks,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	Deploy        interface{}       `yaml:"deploy,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling to handle build field
func (s *Service) UnmarshalYAML(node *yaml.Node) error {
	var raw ServiceWithRaw
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
					config.Args[k] = fmt.Sprintf("%v", val)
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
