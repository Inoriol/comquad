package deploy

import "os"

// IsRootless returns true if the current process is running as a non-root user.
func IsRootless() bool {
	return os.Getuid() != 0
}
