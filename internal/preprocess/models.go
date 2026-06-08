package preprocess

// ComposeFile represents the top-level structure of a docker-compose.yaml
type ComposeFile struct {
        Version  string             `yaml:"version,omitempty"`
        Services map[string]*Service `yaml:"services"`
        Networks map[string]*Network `yaml:"networks,omitempty"`
        Volumes  map[string]*Volume  `yaml:"volumes,omitempty"`
        Config   *ProjectConfig      `yaml:"-"` // Internal use
}

// Service represents a single container service in the compose file
type Service struct {
        ContainerName string            `yaml:"container_name,omitempty"`
        Image         string            `yaml:"image,omitempty"`
        Build         interface{}       `yaml:"build,omitempty"`
        Ports         []string          `yaml:"ports,omitempty"`
        Volumes       []string          `yaml:"volumes,omitempty"`
        Networks      []string          `yaml:"networks,omitempty"`
        Environment   map[string]string `yaml:"environment,omitempty"`
        Labels        map[string]string `yaml:"labels,omitempty"` // Fix: was `yaml:` — incomplete tag, compile error
        Deploy        interface{}       `yaml:"deploy,omitempty"`
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
