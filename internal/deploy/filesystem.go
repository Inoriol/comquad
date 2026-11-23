package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
)

// TargetDirResolver determines the correct Systemd directory based on UID
type TargetDirResolver struct{}

// NewTargetDirResolver creates a new resolver
func NewTargetDirResolver() *TargetDirResolver {
	return &TargetDirResolver{}
}

// GetSystemdPath returns either /etc/containers/systemd or $XDG_CONFIG_HOME/containers/systemd
// (default: ~/.config/containers/systemd)
func (r *TargetDirResolver) GetSystemdPath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	if currentUser.Uid == "0" {
		return "/etc/containers/systemd", nil
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(currentUser.HomeDir, ".config")
	}
	return filepath.Join(configDir, "containers", "systemd"), nil
}

// GetSystemdUnitPath returns the systemd unit directory where .timer files should be placed.
// For root: /etc/systemd/system
// For non-root: $XDG_CONFIG_HOME/systemd/user (default: ~/.config/systemd/user)
func (r *TargetDirResolver) GetSystemdUnitPath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	if currentUser.Uid == "0" {
		return "/etc/systemd/system", nil
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(currentUser.HomeDir, ".config")
	}
	return filepath.Join(configDir, "systemd", "user"), nil
}

// ValidatePodmanVersion checks that podman meets the minimum required version (4.8.0).
// Returns nil if the version is sufficient, or an error describing the shortfall.
func ValidatePodmanVersion() error {
	v, err := DetectPodmanVersion()
	if err != nil {
		return err
	}
	if !v.AtLeast(4, 8) {
		return fmt.Errorf("podman %d.%d detected — comquad requires 4.8+ (for .image quadlet support)", v.Major, v.Minor)
	}
	return nil
}

// DetectPodmanVersion returns the installed podman version, parsed into a
// compose2quadlet.Version. The result is used to gate generated quadlet
// directives on features the installed podman actually supports.
func DetectPodmanVersion() (c2q.Version, error) {
	out, err := exec.Command("podman", "version", "--format", "{{.Version}}").Output()
	if err != nil {
		return c2q.Version{}, fmt.Errorf("failed to detect podman version: %w", err)
	}
	return c2q.ParseVersion(strings.TrimSpace(string(out)))
}

// StateFileExists returns true if the comquad state file already exists on disk.
func StateFileExists() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		stateDir = filepath.Join(home, ".local", "state")
	}
	statePath := filepath.Join(stateDir, "comquad", "projects.json")
	_, err = os.Stat(statePath)
	return err == nil
}

// ResolveRuntimeSecretsDir returns the XDG-compliant runtime secrets directory.
// Rootless: $XDG_RUNTIME_DIR/comquad/secrets (default: /run/user/<uid>/comquad/secrets)
// Rootful: /run/comquad/secrets
func ResolveRuntimeSecretsDir() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	if currentUser.Uid == "0" {
		return "/run/comquad/secrets", nil
	}

	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = filepath.Join("/run/user", currentUser.Uid)
	}
	return filepath.Join(runtimeDir, "comquad", "secrets"), nil
}
