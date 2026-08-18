package mapper

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
	"github.com/compose-spec/compose-go/v2/types"
)

func Service(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	dirs = append(dirs, memory(svc, cfg)...)
	dirs = append(dirs, cpu(svc, cfg, svc.Deploy)...)
	dirs = append(dirs, tasksMax(svc, cfg, svc.Deploy)...)
	dirs = append(dirs, oom(svc, cfg)...)
	dirs = append(dirs, blkio(svc, cfg)...)
	dirs = append(dirs, restart(svc, cfg)...)
	dirs = append(dirs, ulimitsP2(svc, cfg)...)
	dirs = append(dirs, slice(svc, cfg)...)

	return dirs
}

func memory(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if svc.MemLimit > 0 && !cfg.PodmanVersion.AtLeast(5, 5) {
		dirs = append(dirs, c2qtypes.Directive{Key: "MemoryMax", Values: []string{strconv.FormatInt(int64(svc.MemLimit), 10)}})
	}
	if svc.MemReservation > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "MemoryLow", Values: []string{strconv.FormatInt(int64(svc.MemReservation), 10)}})
	}
	if svc.MemSwapLimit > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "MemorySwapMax", Values: []string{strconv.FormatInt(int64(svc.MemSwapLimit), 10)}})
	}

	if svc.Deploy != nil {
		if svc.Deploy.Resources.Limits != nil && svc.Deploy.Resources.Limits.MemoryBytes > 0 {
			dirs = append(dirs, c2qtypes.Directive{Key: "MemoryMax", Values: []string{strconv.FormatInt(int64(svc.Deploy.Resources.Limits.MemoryBytes), 10)}})
		}
		if svc.Deploy.Resources.Reservations != nil && svc.Deploy.Resources.Reservations.MemoryBytes > 0 {
			dirs = append(dirs, c2qtypes.Directive{Key: "MemoryLow", Values: []string{strconv.FormatInt(int64(svc.Deploy.Resources.Reservations.MemoryBytes), 10)}})
		}
	}

	return dirs
}

func cpu(svc types.ServiceConfig, cfg *c2qtypes.Config, deploy *types.DeployConfig) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if svc.CPUS > 0 {
		quota := int(math.Round(float64(svc.CPUS) * 100))
		dirs = append(dirs, c2qtypes.Directive{Key: "CPUQuota", Values: []string{fmt.Sprintf("%d%%", quota)}})
	}
	if svc.CPUShares > 0 {
		weight := max(1, int(svc.CPUShares*100/1024))
		dirs = append(dirs, c2qtypes.Directive{Key: "CPUWeight", Values: []string{strconv.Itoa(weight)}})
	}
	if svc.CPUPeriod > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "CPUQuotaPeriodSec", Values: []string{fmt.Sprintf("%dms", svc.CPUPeriod/1000)}})
	}
	if svc.CPUQuota > 0 && svc.CPUS == 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "CPUQuota", Values: []string{fmt.Sprintf("%d%%", svc.CPUQuota/1000)}})
	}
	if svc.CPUSet != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "AllowedCPUs", Values: []string{svc.CPUSet}})
	}

	if deploy != nil {
		if deploy.Resources.Limits != nil && deploy.Resources.Limits.NanoCPUs > 0 && svc.CPUS == 0 && svc.CPUQuota == 0 {
			quota := int(math.Round(float64(deploy.Resources.Limits.NanoCPUs) * 100 / 1e9))
			dirs = append(dirs, c2qtypes.Directive{Key: "CPUQuota", Values: []string{fmt.Sprintf("%d%%", quota)}})
		}
		if deploy.Resources.Reservations != nil && deploy.Resources.Reservations.NanoCPUs > 0 && svc.CPUShares == 0 {
			weight := max(1, int(math.Round(float64(deploy.Resources.Reservations.NanoCPUs)*100/1e9)))
			dirs = append(dirs, c2qtypes.Directive{Key: "CPUWeight", Values: []string{strconv.Itoa(weight)}})
		}
	}

	return dirs
}

func tasksMax(svc types.ServiceConfig, cfg *c2qtypes.Config, deploy *types.DeployConfig) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if svc.PidsLimit > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "TasksMax", Values: []string{strconv.FormatInt(svc.PidsLimit, 10)}})
	}
	if deploy != nil && deploy.Resources.Limits != nil && deploy.Resources.Limits.Pids > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "TasksMax", Values: []string{strconv.FormatInt(deploy.Resources.Limits.Pids, 10)}})
	}

	return dirs
}

func oom(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if svc.OomScoreAdj != 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "OOMScoreAdjust", Values: []string{strconv.FormatInt(svc.OomScoreAdj, 10)}})
	}

	return dirs
}

func blkio(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	if svc.BlkioConfig == nil {
		return nil
	}
	var dirs []c2qtypes.Directive

	if svc.BlkioConfig.Weight > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "IOWeight", Values: []string{strconv.Itoa(int(svc.BlkioConfig.Weight))}})
	}
	for _, wd := range svc.BlkioConfig.WeightDevice {
		dirs = append(dirs, c2qtypes.Directive{Key: "IODeviceWeight", Values: []string{fmt.Sprintf("%s %d", wd.Path, wd.Weight)}})
	}
	for _, td := range svc.BlkioConfig.DeviceReadBps {
		dirs = append(dirs, c2qtypes.Directive{Key: "IOReadBandwidthMax", Values: []string{fmt.Sprintf("%s %d", td.Path, td.Rate)}})
	}
	for _, td := range svc.BlkioConfig.DeviceWriteBps {
		dirs = append(dirs, c2qtypes.Directive{Key: "IOWriteBandwidthMax", Values: []string{fmt.Sprintf("%s %d", td.Path, td.Rate)}})
	}
	for _, td := range svc.BlkioConfig.DeviceReadIOps {
		dirs = append(dirs, c2qtypes.Directive{Key: "IOReadIOPSMax", Values: []string{fmt.Sprintf("%s %d", td.Path, td.Rate)}})
	}
	for _, td := range svc.BlkioConfig.DeviceWriteIOps {
		dirs = append(dirs, c2qtypes.Directive{Key: "IOWriteIOPSMax", Values: []string{fmt.Sprintf("%s %d", td.Path, td.Rate)}})
	}

	return dirs
}

func restart(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if svc.Deploy != nil && svc.Deploy.RestartPolicy != nil {
		dirs = append(dirs, deployRestart(svc.Deploy.RestartPolicy)...)
	} else if restartDirective(svc.Restart) {
		dirs = append(dirs, resolveRestart(svc.Restart)...)
	}

	return dirs
}

func restartDirective(restart string) bool {
	return restart != "" && restart != "no"
}

func resolveRestart(restart string) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive
	switch {
	case restart == "always":
		dirs = append(dirs, c2qtypes.Directive{Key: "Restart", Values: []string{"always"}})
	case restart == "unless-stopped":
		dirs = append(dirs, c2qtypes.Directive{Key: "Restart", Values: []string{"always"}})
	case restart == "on-failure":
		dirs = append(dirs, c2qtypes.Directive{Key: "Restart", Values: []string{"on-failure"}})
	case strings.HasPrefix(restart, "on-failure:"):
		dirs = append(dirs, c2qtypes.Directive{Key: "Restart", Values: []string{"on-failure"}})
		if n := strings.TrimPrefix(restart, "on-failure:"); n != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "StartLimitBurst", Values: []string{n}})
		}
	default:
		return nil
	}
	return dirs
}

func deployRestart(rp *types.RestartPolicy) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	switch rp.Condition {
	case "any", "always":
		dirs = append(dirs, c2qtypes.Directive{Key: "Restart", Values: []string{"always"}})
	case "on-failure":
		dirs = append(dirs, c2qtypes.Directive{Key: "Restart", Values: []string{"on-failure"}})
	case "none", "no":
		dirs = append(dirs, c2qtypes.Directive{Key: "Restart", Values: []string{"no"}})
	default:
		return nil
	}
	if rp.Delay != nil {
		secs := max(1, int(math.Round(float64(*rp.Delay)/1e9)))
		dirs = append(dirs, c2qtypes.Directive{Key: "RestartSec", Values: []string{strconv.Itoa(secs)}})
	}
	if rp.MaxAttempts != nil && *rp.MaxAttempts > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "StartLimitBurst", Values: []string{strconv.FormatUint(*rp.MaxAttempts, 10)}})
		dirs = append(dirs, c2qtypes.Directive{Key: "StartLimitIntervalSec", Values: []string{"0"}})
	}
	if rp.Window != nil {
		secs := max(1, int(math.Round(float64(*rp.Window)/1e9)))
		dirs = append(dirs, c2qtypes.Directive{Key: "RuntimeMaxSec", Values: []string{strconv.Itoa(secs)}})
	}

	return dirs
}

func ulimitsP2(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	if len(svc.Ulimits) == 0 {
		return nil
	}
	var dirs []c2qtypes.Directive
	for _, name := range sortedKeys(svc.Ulimits) {
		u := svc.Ulimits[name]
		key := "Limit" + strings.ToUpper(name)
		var val string
		if u.Single > 0 {
			val = strconv.Itoa(u.Single)
		} else {
			val = strconv.Itoa(u.Soft) + ":" + strconv.Itoa(u.Hard)
		}
		dirs = append(dirs, c2qtypes.Directive{Key: key, Values: []string{val}})
	}
	return dirs
}

func slice(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	if svc.Cgroup == "" || svc.Cgroup == "host" || svc.Cgroup == "private" {
		return nil
	}
	return []c2qtypes.Directive{{Key: "Slice", Values: []string{svc.Cgroup}}}
}
