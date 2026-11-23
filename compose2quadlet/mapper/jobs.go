package mapper

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func JobContainer(job types.JobConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if job.Build != nil {
		if !cfg.PodmanVersion.AtLeast(5, 2) && !cfg.PseudoQuadlet {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningFatal,
				Service: job.Name,
				Field:   "build",
				Message: "requires podman >= 5.2.0",
				Since:   "5.2.0",
			})
		}
		if cfg.PodmanVersion.AtLeast(5, 2) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{job.Name + ".build"}})
		} else if cfg.PseudoQuadlet {
			img := jobBuildImageTag(job, cfg)
			dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{img}})
		} else if job.Image != "" {
			if cfg.PodmanVersion.AtLeast(4, 8) {
				dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{job.Name + ".image"}})
			} else {
				dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{job.Image}})
			}
		}
	} else if cfg.PodmanVersion.AtLeast(4, 8) {
		if job.Image != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{job.Name + ".image"}})
		}
	} else if job.Image != "" {
		if cfg.PseudoQuadlet {
			img := job.Image
			if !cfg.SkipImageNormalization {
				img = normalizeImage(img)
			}
			dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{img}})
		} else {
			dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{job.Image}})
		}
	}

	if len(job.Entrypoint) > 0 {
		if cfg.PodmanVersion.AtLeast(5, 0) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Entrypoint", Values: []string{strings.Join(job.Entrypoint, " ")}})
		} else {
			dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--entrypoint " + strings.Join(job.Entrypoint, " ")}})
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningDegraded,
				Service: job.Name,
				Field:   "entrypoint",
				Message: "using PodmanArgs fallback; upgrade to podman >= 5.0.0 for native Entrypoint= support",
				Since:   "5.0.0",
			})
		}
	}

	if job.Init != nil && *job.Init {
		dirs = append(dirs, c2qtypes.Directive{Key: "RunInit", Values: []string{"true"}})
	}
	if job.ReadOnly {
		dirs = append(dirs, c2qtypes.Directive{Key: "ReadOnly", Values: []string{"true"}})
	}
	if job.User != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "User", Values: []string{job.User}})
	}
	if job.GroupAdd != nil {
		if cfg.PodmanVersion.AtLeast(5, 1) {
			for _, g := range job.GroupAdd {
				dirs = append(dirs, c2qtypes.Directive{Key: "GroupAdd", Values: []string{g}})
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "group_add",
				Message: "requires podman >= 5.1.0",
				Since:   "5.1.0",
			})
		}
	}
	if job.WorkingDir != "" {
		if cfg.PodmanVersion.AtLeast(4, 6) {
			dirs = append(dirs, c2qtypes.Directive{Key: "WorkingDir", Values: []string{job.WorkingDir}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "working_dir",
				Message: "requires podman >= 4.6.0",
				Since:   "4.6.0",
			})
		}
	}

	dirs = append(dirs, SecurityOpts(job.SecurityOpt, job.Name, cfg)...)

	if job.MemLimit > 0 {
		if cfg.PodmanVersion.AtLeast(5, 5) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Memory", Values: []string{strconv.FormatInt(int64(job.MemLimit), 10)}})
		}
	}
	if job.Cgroup == "host" {
		if cfg.PodmanVersion.AtLeast(5, 3) {
			dirs = append(dirs, c2qtypes.Directive{Key: "CgroupsMode", Values: []string{"host"}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "cgroup",
				Message: "requires podman >= 5.3.0",
				Since:   "5.3.0",
			})
		}
	}
	if job.Cgroup == "private" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--cgroupns private"}})
	}

	if job.UserNSMode != "" {
		if cfg.PodmanVersion.AtLeast(4, 5) {
			dirs = append(dirs, c2qtypes.Directive{Key: "UserNS", Values: []string{job.UserNSMode}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "userns_mode",
				Message: "requires podman >= 4.5.0",
				Since:   "4.5.0",
			})
		}
	}
	if job.Hostname != "" {
		if cfg.PodmanVersion.AtLeast(4, 6) {
			dirs = append(dirs, c2qtypes.Directive{Key: "HostName", Values: []string{job.Hostname}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "hostname",
				Message: "requires podman >= 4.6.0",
				Since:   "4.6.0",
			})
		}
	}
	if job.ShmSize != 0 {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			dirs = append(dirs, c2qtypes.Directive{Key: "ShmSize", Values: []string{strconv.FormatInt(int64(job.ShmSize), 10)}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "shm_size",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}
	if len(job.Sysctls) > 0 {
		if cfg.PodmanVersion.AtLeast(4, 6) {
			sysKeys := sortedKeys(job.Sysctls)
			for _, k := range sysKeys {
				dirs = append(dirs, c2qtypes.Directive{Key: "Sysctl", Values: []string{k + "=" + job.Sysctls[k]}})
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "sysctls",
				Message: "requires podman >= 4.6.0",
				Since:   "4.6.0",
			})
		}
	}
	if len(job.DNS) > 0 {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			for _, d := range job.DNS {
				dirs = append(dirs, c2qtypes.Directive{Key: "DNS", Values: []string{d}})
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "dns",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}
	if len(job.DNSSearch) > 0 {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			for _, d := range job.DNSSearch {
				dirs = append(dirs, c2qtypes.Directive{Key: "DNSSearch", Values: []string{d}})
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "dns_search",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}
	for _, d := range job.DNSOpts {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			dirs = append(dirs, c2qtypes.Directive{Key: "DNSOption", Values: []string{d}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "dns_opt",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}
	if job.StopGracePeriod != nil {
		if cfg.PodmanVersion.AtLeast(5, 0) {
			dirs = append(dirs, c2qtypes.Directive{Key: "StopTimeout", Values: []string{time.Duration(*job.StopGracePeriod).String()}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "stop_grace_period",
				Message: "requires podman >= 5.0.0",
				Since:   "5.0.0",
			})
		}
	}
	if job.StopSignal != "" {
		if cfg.PodmanVersion.AtLeast(5, 2) {
			dirs = append(dirs, c2qtypes.Directive{Key: "StopSignal", Values: []string{job.StopSignal}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "stop_signal",
				Message: "requires podman >= 5.2.0",
				Since:   "5.2.0",
			})
		}
	}
	if job.PullPolicy != "" {
		if cfg.PodmanVersion.AtLeast(4, 6) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Pull", Values: []string{job.PullPolicy}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "pull_policy",
				Message: "requires podman >= 4.6.0",
				Since:   "4.6.0",
			})
		}
	}

	for _, k := range sortedKeys(job.Labels) {
		dirs = append(dirs, c2qtypes.Directive{Key: "Label", Values: []string{fmt.Sprintf("%s=%s", k, job.Labels[k])}})
	}
	for _, k := range sortedKeys(job.Annotations) {
		dirs = append(dirs, c2qtypes.Directive{Key: "Annotation", Values: []string{fmt.Sprintf("%s=%s", k, job.Annotations[k])}})
	}
	for _, c := range job.CapAdd {
		dirs = append(dirs, c2qtypes.Directive{Key: "AddCapability", Values: []string{c}})
	}
	for _, c := range job.CapDrop {
		dirs = append(dirs, c2qtypes.Directive{Key: "DropCapability", Values: []string{c}})
	}
	for _, e := range job.Expose {
		dirs = append(dirs, c2qtypes.Directive{Key: "ExposeHostPort", Values: []string{e}})
	}

	for _, dev := range job.Devices {
		if dev.Target == "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "AddDevice", Values: []string{dev.Source}})
		} else if dev.Permissions != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "AddDevice", Values: []string{dev.Source + ":" + dev.Target + ":" + dev.Permissions}})
		} else {
			dirs = append(dirs, c2qtypes.Directive{Key: "AddDevice", Values: []string{dev.Source + ":" + dev.Target}})
		}
	}

	dirs = append(dirs, jobT1Container(job, cfg)...)
	dirs = append(dirs, jobT3Container(job, cfg)...)

	return dirs
}

func jobT1Container(job types.JobConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if len(job.Command) > 0 {
		var quoted []string
		for _, cmd := range job.Command {
			quoted = append(quoted, "'"+cmd+"'")
		}
		dirs = append(dirs, c2qtypes.Directive{Key: "Exec", Values: []string{strings.Join(quoted, " ")}})
	}

	for _, k := range sortedKeys(job.Environment) {
		v := job.Environment[k]
		if v != nil {
			dirs = append(dirs, c2qtypes.Directive{Key: "Environment", Values: []string{k + "=" + *v}})
		} else {
			if cfg.PodmanVersion.AtLeast(5, 6) {
				dirs = append(dirs, c2qtypes.Directive{Key: "Environment", Values: []string{k}})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: job.Name,
					Field:   "environment",
					Message: "bare key requires podman >= 5.6.0",
					Since:   "5.6.0",
				})
			}
		}
	}
	for _, f := range job.EnvFiles {
		dirs = append(dirs, c2qtypes.Directive{Key: "EnvironmentFile", Values: []string{f.Path}})
	}
	for _, t := range job.Tmpfs {
		dirs = append(dirs, c2qtypes.Directive{Key: "Tmpfs", Values: []string{t}})
	}

	if job.Logging != nil {
		if job.Logging.Driver != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "LogDriver", Values: []string{job.Logging.Driver}})
		}
		if len(job.Logging.Options) > 0 {
			if cfg.PodmanVersion.AtLeast(5, 2) {
				for _, k := range sortedKeys(job.Logging.Options) {
					dirs = append(dirs, c2qtypes.Directive{Key: "LogOpt", Values: []string{k + "=" + job.Logging.Options[k]}})
				}
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: job.Name,
					Field:   "logging.options",
					Message: "requires podman >= 5.2.0",
					Since:   "5.2.0",
				})
			}
		}
	} else {
		if job.LogDriver != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "LogDriver", Values: []string{job.LogDriver}})
		}
		for _, k := range sortedKeys(job.LogOpt) {
			if cfg.PodmanVersion.AtLeast(5, 2) {
				dirs = append(dirs, c2qtypes.Directive{Key: "LogOpt", Values: []string{k + "=" + job.LogOpt[k]}})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: job.Name,
					Field:   "logging.options",
					Message: "requires podman >= 5.2.0",
					Since:   "5.2.0",
				})
			}
		}
	}

	if job.PidsLimit != 0 {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			dirs = append(dirs, c2qtypes.Directive{Key: "PidsLimit", Values: []string{strconv.FormatInt(job.PidsLimit, 10)}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "pids_limit",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}

	if len(job.ExtraHosts) > 0 {
		if cfg.PodmanVersion.AtLeast(5, 3) {
			for _, host := range sortedKeys(job.ExtraHosts) {
				for _, ip := range job.ExtraHosts[host] {
					dirs = append(dirs, c2qtypes.Directive{Key: "AddHost", Values: []string{host + ":" + ip}})
				}
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "extra_hosts",
				Message: "requires podman >= 5.3.0",
				Since:   "5.3.0",
			})
		}
	}

	for _, v := range job.Volumes {
		if v.Type == types.VolumeTypeBind {
			dirs = append(dirs, c2qtypes.Directive{Key: "Mount", Values: []string{formatJobBindMount(v, cfg.WorkingDirectory)}})
		} else if v.Type == types.VolumeTypeTmpfs {
			dirs = append(dirs, c2qtypes.Directive{Key: "Tmpfs", Values: []string{formatJobTmpfsMount(v)}})
		} else {
			vol := v.String()
			if colon := strings.IndexByte(vol, ':'); colon > 0 {
				vol = vol[:colon] + ".volume" + vol[colon:]
			}
			dirs = append(dirs, c2qtypes.Directive{Key: "Volume", Values: []string{vol}})
		}
	}

	for _, p := range job.Ports {
		dirs = append(dirs, c2qtypes.Directive{Key: "PublishPort", Values: []string{formatJobPort(p)}})
	}

	if job.NetworkMode == "host" {
		dirs = append(dirs, c2qtypes.Directive{Key: "Network", Values: []string{"host"}})
	} else if job.NetworkMode == "none" {
		dirs = append(dirs, c2qtypes.Directive{Key: "Network", Values: []string{"none"}})
	} else if strings.HasPrefix(job.NetworkMode, "service:") {
		target := strings.TrimPrefix(job.NetworkMode, "service:")
		if cfg.PodmanVersion.AtLeast(5, 3) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Network", Values: []string{"container:" + target + ".container"}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: job.Name,
				Field:   "network_mode",
				Message: "network_mode: service:<name> requires podman >= 5.3.0",
				Since:   "5.3.0",
			})
		}
	} else if job.NetworkMode == "" || job.NetworkMode == "bridge" {
		for _, name := range sortedKeys(job.Networks) {
			net := job.Networks[name]
			dirs = append(dirs, c2qtypes.Directive{Key: "Network", Values: []string{name + ".network"}})
			if net == nil {
				continue
			}
			if net.Ipv4Address != "" {
				dirs = append(dirs, c2qtypes.Directive{Key: "IP", Values: []string{net.Ipv4Address}})
			}
			if net.Ipv6Address != "" {
				dirs = append(dirs, c2qtypes.Directive{Key: "IP6", Values: []string{net.Ipv6Address}})
			}
		}
	}

	return dirs
}

func jobT3Container(job types.JobConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if job.Tty {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--tty"}})
	}
	if job.StdinOpen {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--attach stdin"}})
	}
	if job.Runtime != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "GlobalArgs", Values: []string{"--runtime " + job.Runtime}})
	}
	if job.MacAddress != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--mac-address " + job.MacAddress}})
	}
	if job.Ipc == "shareable" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--ipc shareable"}})
	}
	if job.Pid == "host" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--pid host"}})
	}
	if job.Uts == "host" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--uts host"}})
	}
	if job.Privileged {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--privileged"}})
	}
	if job.MemSwappiness > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{fmt.Sprintf("--memory-swappiness %d", job.MemSwappiness)}})
	}
	if job.CPURTRuntime > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{fmt.Sprintf("--cpu-rt-runtime %d", job.CPURTRuntime)}})
	}
	if job.CPURTPeriod > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{fmt.Sprintf("--cpu-rt-period %d", job.CPURTPeriod)}})
	}
	for _, rule := range job.DeviceCgroupRules {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--device-cgroup-rule " + rule}})
	}
	for _, k := range sortedKeys(job.StorageOpt) {
		dirs = append(dirs, c2qtypes.Directive{Key: "GlobalArgs", Values: []string{"--storage-opt " + k + "=" + job.StorageOpt[k]}})
	}
	if job.OomKillDisable {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--oom-kill-disable"}})
	}
	if job.CgroupParent != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--cgroup-parent " + job.CgroupParent}})
	}
	return dirs
}

func formatJobBindMount(v types.ServiceVolumeConfig, workDir string) string {
	var parts []string
	parts = append(parts, "type=bind")
	source := v.Source
	if workDir != "" && !filepath.IsAbs(source) {
		source = filepath.Join(workDir, source)
	}
	parts = append(parts, "source="+source)
	parts = append(parts, "destination="+v.Target)
	if v.ReadOnly {
		parts = append(parts, "readonly")
	}
	if v.Bind != nil {
		if v.Bind.Propagation != "" {
			parts = append(parts, "bind-propagation="+v.Bind.Propagation)
		}
		if v.Bind.SELinux != "" {
			parts = append(parts, jobSelinuxToRelabel(v.Bind.SELinux))
		}
	}
	return strings.Join(parts, ",")
}

func jobSelinuxToRelabel(selinux string) string {
	if selinux == "Z" {
		return "relabel=private"
	}
	return "relabel=shared"
}

func formatJobTmpfsMount(v types.ServiceVolumeConfig) string {
	val := v.Target
	if v.Tmpfs != nil {
		var opts []string
		if v.Tmpfs.Size > 0 {
			opts = append(opts, fmt.Sprintf("size=%d", v.Tmpfs.Size))
		}
		if v.Tmpfs.Mode > 0 {
			opts = append(opts, fmt.Sprintf("mode=%o", v.Tmpfs.Mode))
		}
		if len(opts) > 0 {
			val += ":" + strings.Join(opts, ",")
		}
	}
	return val
}

func formatJobPort(p types.ServicePortConfig) string {
	target := strconv.FormatUint(uint64(p.Target), 10)
	if p.Published == "" {
		return target
	}
	var result string
	if p.Protocol != "" {
		result = p.Published + ":" + target + "/" + p.Protocol
	} else {
		result = p.Published + ":" + target
	}
	if p.HostIP != "" {
		result = p.HostIP + ":" + result
	}
	return result
}

func jobBuildImageTag(job types.JobConfig, cfg *c2qtypes.Config) string {
	if job.Build != nil && len(job.Build.Tags) > 0 {
		return job.Build.Tags[0]
	}
	tag := job.Name + ":latest"
	if cfg.ProjectName != "" {
		tag = cfg.ProjectName + "_" + job.Name + ":latest"
	}
	return tag
}

func JobUnit(job types.JobConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	for name := range job.DependsOn {
		dep := job.DependsOn[name]
		if dep.Condition == "service_started" || dep.Condition == "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "After", Values: []string{name + ".container"}})
			dirs = append(dirs, c2qtypes.Directive{Key: "Requires", Values: []string{name + ".container"}})
		}
	}

	return dirs
}

func JobImages(jobs types.Jobs, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	var units []c2qtypes.QuadletUnit
	for name, job := range jobs {
		if job.Image == "" {
			continue
		}
		var dirs []c2qtypes.Directive
		img := job.Image
		if !cfg.SkipImageNormalization {
			img = normalizeImage(img)
		}
		dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{img}})

		useNative := cfg.PodmanVersion.AtLeast(4, 8)
		if !useNative && !cfg.PseudoQuadlet {
			continue
		}

		if job.PullPolicy != "" {
			policy := composePolicyToQuadlet(job.PullPolicy)
			if policy != "" {
				if cfg.PodmanVersion.AtLeast(5, 6) {
					dirs = append(dirs, c2qtypes.Directive{Key: "Policy", Values: []string{policy}})
				} else {
					cfg.Warn(c2qtypes.Warning{
						Level:   c2qtypes.WarningSkipped,
						Service: name,
						Field:   "pull_policy",
						Message: "Policy= in .image requires podman >= 5.6.0",
						Since:   "5.6.0",
					})
				}
			}
		}

		if job.Platform != "" {
			if useNative {
				parts := strings.SplitN(job.Platform, "/", 3)
				if len(parts) >= 1 && parts[0] != "" {
					dirs = append(dirs, c2qtypes.Directive{Key: "OS", Values: []string{parts[0]}})
				}
				if len(parts) >= 2 && parts[1] != "" {
					dirs = append(dirs, c2qtypes.Directive{Key: "Arch", Values: []string{parts[1]}})
				}
				if len(parts) >= 3 && parts[2] != "" {
					dirs = append(dirs, c2qtypes.Directive{Key: "Variant", Values: []string{parts[2]}})
				}
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "platform",
					Message: "platform not supported in pseudo-image mode",
				})
			}
		}

		unitType := c2qtypes.UnitImage
		if !useNative {
			unitType = c2qtypes.UnitCQImage
		}

		units = append(units, c2qtypes.QuadletUnit{
			Type:     unitType,
			Name:     name,
			Sections: []c2qtypes.Section{{Name: c2qtypes.SectionImage, Directives: dirs}},
		})
	}
	return units
}

func JobBuilds(jobs types.Jobs, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	var units []c2qtypes.QuadletUnit
	for name, job := range jobs {
		if job.Build == nil {
			continue
		}
		useNative := cfg.PodmanVersion.AtLeast(5, 2)
		if !useNative && !cfg.PseudoQuadlet {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningFatal,
				Service: name,
				Field:   "build",
				Message: "build requires podman >= 5.2.0 or pseudo-quadlet enabled",
				Since:   "5.2.0",
			})
			continue
		}
		build := job.Build
		var dirs []c2qtypes.Directive

		if build.Context != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "SetWorkingDirectory", Values: []string{build.Context}})
		}

		if build.Dockerfile != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "File", Values: []string{build.Dockerfile}})
		} else if build.DockerfileInline != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "File", Values: []string{build.DockerfileInline}})
		}
		if build.Target != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "Target", Values: []string{build.Target}})
		}
		if build.Network != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "Network", Values: []string{build.Network}})
		}
		if build.NoCache {
			dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--no-cache"}})
		}
		for _, k := range sortedKeys(build.Labels) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Label", Values: []string{fmt.Sprintf("%s=%s", k, build.Labels[k])}})
		}
		for _, tag := range build.Tags {
			dirs = append(dirs, c2qtypes.Directive{Key: "ImageTag", Values: []string{tag}})
		}
		if len(build.Tags) == 0 {
			tag := name + ":latest"
			if cfg.ProjectName != "" {
				tag = cfg.ProjectName + "_" + name + ":latest"
			}
			dirs = append(dirs, c2qtypes.Directive{Key: "ImageTag", Values: []string{tag}})
		}

		unitType := c2qtypes.UnitBuild
		if !useNative {
			unitType = c2qtypes.UnitCQBuild
		}

		units = append(units, c2qtypes.QuadletUnit{
			Type:     unitType,
			Name:     name,
			Sections: []c2qtypes.Section{{Name: c2qtypes.SectionBuild, Directives: dirs}},
		})
	}
	return units
}
