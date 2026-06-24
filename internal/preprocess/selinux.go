package preprocess

import (
	"os"
	"strings"
)

// selinuxEnabled is a package-level variable that controls SELinux detection.
// It defaults to nil (use filesystem detection) and can be overridden in tests.
var selinuxEnabled *bool

// selinuxMode is a package-level variable that stores the detected SELinux mode.
// It defaults to "" (use filesystem detection) and can be overridden in tests.
var selinuxMode string

// checkSELinux checks whether SELinux is enabled and returns its mode.
// It checks for the existence of /sys/fs/selinux and reads the enforcement mode.
func checkSELinux() (enabled bool, mode string) {
	if _, err := os.Stat("/sys/fs/selinux"); os.IsNotExist(err) {
		return false, "Disabled"
	}

	data, err := os.ReadFile("/sys/fs/selinux/enforce")
	if err != nil {
		return false, "Disabled"
	}

	content := strings.TrimSpace(string(data))
	if content == "1" {
		return true, "Enforcing"
	} else if content == "0" {
		return true, "Permissive"
	}

	return false, "Unknown"
}

// IsSELinuxEnabled returns true if SELinux is detected as enabled on the host.
// It caches the result so subsequent calls avoid repeated filesystem access.
func IsSELinuxEnabled() bool {
	if selinuxEnabled != nil {
		return *selinuxEnabled
	}

	enabled, _ := checkSELinux()
	selinuxEnabled = &enabled
	return enabled
}

// SELinuxMode returns the detected SELinux mode string.
// Possible values: "Enforcing", "Permissive", "Disabled", "Unknown".
func SELinuxMode() string {
	if selinuxMode != "" {
		return selinuxMode
	}

	_, mode := checkSELinux()
	selinuxMode = mode
	return mode
}

// SetSELinuxOverrides overrides the SELinux detection result for testing.
// Pass nil for enabled to restore filesystem-based detection.
func SetSELinuxOverrides(enabled *bool, mode string) {
	selinuxEnabled = enabled
	selinuxMode = mode
}
