package types

import (
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	SelinuxContext      bool
	FilePrefix          string
	DefaultNetwork      bool
	PortOffset          int
	ProjectName         string
	Labels              map[string]string
	AutoUpdate          bool
	InstallSection      bool
	NetworkAliases      bool
	PodmanVersion       Version
	Warnings            []Warning
	ImageRetry          int
	ImageRetryDelay     int
	WorkingDirectory    string
	SecretsDir          string
	DryRun              bool
	BuildCacheDir       string
	NormalizeDockerfile bool
	Info                func(string)
	PatchedDockerfiles  map[string]string
}

type Version struct {
	Major, Minor, Patch int
}

func (v Version) AtLeast(major, minor int) bool {
	if v.Major == 0 && v.Minor == 0 {
		return true
	}
	if v.Major > major {
		return true
	}
	if v.Major == major && v.Minor >= minor {
		return true
	}
	return false
}

func DefaultConfig() *Config {
	return &Config{
		SelinuxContext:  true,
		FilePrefix:      "cq-",
		DefaultNetwork:  true,
		PortOffset:      0,
		AutoUpdate:      false,
		InstallSection:  true,
		NetworkAliases:  true,
		ImageRetry:      3,
		ImageRetryDelay: 5,
	}
}

type Option func(*Config)

func WithoutSELinux() Option {
	return func(c *Config) { c.SelinuxContext = false }
}

func WithoutPrefix() Option {
	return func(c *Config) { c.FilePrefix = "" }
}

func WithPrefix(prefix string) Option {
	return func(c *Config) { c.FilePrefix = prefix }
}

func WithoutDefaultNetwork() Option {
	return func(c *Config) { c.DefaultNetwork = false }
}

func WithPortOffset(offset int) Option {
	return func(c *Config) { c.PortOffset = offset }
}

func WithProjectName(name string) Option {
	return func(c *Config) { c.ProjectName = name }
}

func WithLabels(labels map[string]string) Option {
	return func(c *Config) { c.Labels = labels }
}

func WithAutoUpdate() Option {
	return func(c *Config) { c.AutoUpdate = true }
}

func WithoutInstallSection() Option {
	return func(c *Config) { c.InstallSection = false }
}

func WithoutNetworkAliases() Option {
	return func(c *Config) { c.NetworkAliases = false }
}

func WithPodmanVersion(v Version) Option {
	return func(c *Config) { c.PodmanVersion = v }
}

func WithImageRetry(n int) Option {
	return func(c *Config) { c.ImageRetry = n }
}

func WithImageRetryDelay(seconds int) Option {
	return func(c *Config) { c.ImageRetryDelay = seconds }
}

func WithWorkingDirectory(path string) Option {
	return func(c *Config) { c.WorkingDirectory = path }
}

func WithSecretsDirectory(path string) Option {
	return func(c *Config) { c.SecretsDir = path }
}

func WithDryRun() Option {
	return func(c *Config) { c.DryRun = true }
}

func WithBuildCacheDir(path string) Option {
	return func(c *Config) { c.BuildCacheDir = path }
}

func WithDockerfileNormalization() Option {
	return func(c *Config) { c.NormalizeDockerfile = true }
}

func WithInfo(fn func(string)) Option {
	return func(c *Config) { c.Info = fn }
}

func (c *Config) Warn(w Warning) {
	c.Warnings = append(c.Warnings, w)
}

func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) < 2 {
		return Version{}, fmt.Errorf("invalid version %q: expected major.minor[.patch]", s)
	}
	v := Version{}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}
	v.Major = major
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}
	v.Minor = minor
	if len(parts) == 3 {
		v.Patch, _ = strconv.Atoi(parts[2])
	}
	return v, nil
}
