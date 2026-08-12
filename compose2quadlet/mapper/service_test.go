package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestService_MemoryMax(t *testing.T) {
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 4}
	svc := types.ServiceConfig{Name: "web", MemLimit: 1_073_741_824}
	dirs := Service(svc, cfg)
	assertDirective(t, dirs, "MemoryMax", "1073741824")
}

func TestService_MemoryLow(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", MemReservation: 536_870_912}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "MemoryLow", "536870912")
}

func TestService_MemorySwapMax(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", MemSwapLimit: 2_147_483_648}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "MemorySwapMax", "2147483648")
}

func TestService_CPUQuota(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", CPUS: 2.5}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "CPUQuota", "250%")
}

func TestService_CPUWeight(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", CPUShares: 512}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "CPUWeight", "50")
}

func TestService_CPUQuotaPeriodSec(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", CPUPeriod: 50000}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "CPUQuotaPeriodSec", "50ms")
}

func TestService_CPUSet(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", CPUSet: "0,1"}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "AllowedCPUs", "0,1")
}

func TestService_TasksMax(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", PidsLimit: 500}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "TasksMax", "500")
}

func TestService_OOMKillDisable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", OomKillDisable: true}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "ManagedOOMMemoryPressure", "kill")
}

func TestService_OOMScoreAdjust(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", OomScoreAdj: 500}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "OOMScoreAdjust", "500")
}

func TestService_Blkio(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", BlkioConfig: &types.BlkioConfig{Weight: 500}}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "IOWeight", "500")
}

func TestService_BlkioDevices(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", BlkioConfig: &types.BlkioConfig{
		WeightDevice:    []types.WeightDevice{{Path: "/dev/sda", Weight: 100}},
		DeviceReadBps:   []types.ThrottleDevice{{Path: "/dev/sda", Rate: 1048576}},
		DeviceWriteBps:  []types.ThrottleDevice{{Path: "/dev/sdb", Rate: 2097152}},
		DeviceReadIOps:  []types.ThrottleDevice{{Path: "/dev/sdc", Rate: 1000}},
		DeviceWriteIOps: []types.ThrottleDevice{{Path: "/dev/sdd", Rate: 2000}},
	}}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "IODeviceWeight", "/dev/sda 100")
	assertDirective(t, dirs, "IOReadBandwidthMax", "/dev/sda 1048576")
	assertDirective(t, dirs, "IOWriteBandwidthMax", "/dev/sdb 2097152")
	assertDirective(t, dirs, "IOReadIOPSMax", "/dev/sdc 1000")
	assertDirective(t, dirs, "IOWriteIOPSMax", "/dev/sdd 2000")
}

func TestService_Restart_Always(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Restart: "always"}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "Restart", "always")
}

func TestService_Restart_OnFailure(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Restart: "on-failure"}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "Restart", "on-failure")
}

func TestService_Restart_OnFailureWithCount(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Restart: "on-failure:3"}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "Restart", "on-failure")
	assertDirective(t, dirs, "StartLimitBurst", "3")
}

func TestService_Restart_UnlessStopped(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Restart: "unless-stopped"}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "Restart", "always")
}

func TestService_Restart_No(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Restart: "no"}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	if len(dirs) > 0 {
		t.Fatal("should not emit restart directives for restart: no")
	}
}

func TestService_DeployRestart(t *testing.T) {
	delay := types.Duration(10_000_000_000)
	maxAttempts := uint64(5)
	window := types.Duration(600_000_000_000)
	svc := types.ServiceConfig{Name: "web", Deploy: &types.DeployConfig{
		RestartPolicy: &types.RestartPolicy{
			Condition:   "on-failure",
			Delay:       &delay,
			MaxAttempts: &maxAttempts,
			Window:      &window,
		},
	}}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "Restart", "on-failure")
	assertDirective(t, dirs, "RestartSec", "10")
	assertDirective(t, dirs, "StartLimitBurst", "5")
	assertDirective(t, dirs, "RuntimeMaxSec", "600")
}

func TestService_DeployResources(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Deploy: &types.DeployConfig{
		Resources: types.Resources{
			Limits:       &types.Resource{NanoCPUs: 2e9, MemoryBytes: 1_073_741_824, Pids: 100},
			Reservations: &types.Resource{NanoCPUs: 5e8, MemoryBytes: 536_870_912},
		},
	}}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "CPUQuota", "200%")
	assertDirective(t, dirs, "MemoryMax", "1073741824")
	assertDirective(t, dirs, "TasksMax", "100")
	assertDirective(t, dirs, "CPUWeight", "50")
	assertDirective(t, dirs, "MemoryLow", "536870912")
}

func TestService_Ulimits_P2(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Ulimits: map[string]*types.UlimitsConfig{
		"nofile": {Soft: 1024, Hard: 2048},
		"nproc":  {Single: 65535},
	}}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "LimitNOFILE", "1024:2048")
	assertDirective(t, dirs, "LimitNPROC", "65535")
}

func TestService_Slice(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Cgroup: "/system.slice/myapp.slice"}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	assertDirective(t, dirs, "Slice", "/system.slice/myapp.slice")
}

func TestService_Slice_SkipHost(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Cgroup: "host"}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	if hasDirective(dirs, "Slice", "") {
		t.Fatal("should not emit Slice for cgroup: host")
	}
}

func TestService_Noop(t *testing.T) {
	svc := types.ServiceConfig{Name: "web"}
	dirs := Service(svc, c2qtypes.DefaultConfig())
	if len(dirs) > 0 {
		t.Fatalf("expected no directives for empty service, got %v", dirs)
	}
}
