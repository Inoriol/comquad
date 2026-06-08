package deploy

import (
        "fmt"
        "os/user"
        "path/filepath"
        // "os" ❌ removed — never used in this file
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
                // Fix 1: %d is for integers, %w correctly wraps the error value
                return "", fmt.Errorf("failed to get current user: %w", err)
        }

        // Check if UID is 0 (root)
        // Note: Uid is a string type in os/user, so "0" comparison is correct
        if currentUser.Uid == "0" {
                return "/etc/containers/systemd", nil
        }

        // For non-root users, use ~/.config/containers/systemd
        return filepath.Join(currentUser.HomeDir, ".config", "containers", "systemd"), nil
}
