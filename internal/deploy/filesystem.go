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

// GetSystemdPath returns either /etc/containers/systemd or ~/.config/containers/systemd
func (r *TargetDirResolver) GetSystemdPath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	// Check if UID is 0 (root)
	if currentUser.Uid == "0" {
		return "/etc/containers/systemd", nil
	}

	// For non-root users, use ~/.config/containers/systemd
	return filepath.Join(currentUser.HomeDir, ".config", "containers", "systemd"), nil
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
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(home, ".local", "share")
	}
	statePath := filepath.Join(dataDir, "comquad", "projects.json")
	_, err = os.Stat(statePath)
	return err == nil
}
