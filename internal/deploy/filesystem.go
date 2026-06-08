package deploy

import (
	"fmt"
	"os/user"
	"path/filepath"
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
