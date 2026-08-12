package deploy

import (
	"os"
	"strings"
	"sync"
)

var (
	selinuxOnce    sync.Once
	selinuxEnabled bool
	selinuxRead    bool
)

func IsSELinuxEnabled() bool {
	selinuxOnce.Do(func() {
		data, err := os.ReadFile("/sys/fs/selinux/enforce")
		if err != nil {
			return
		}
		selinuxRead = true
		selinuxEnabled = strings.TrimSpace(string(data)) == "1"
	})
	return selinuxEnabled
}

func IsSELinuxAvailable() bool {
	IsSELinuxEnabled()
	return selinuxRead
}
