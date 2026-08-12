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

func Container(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive
	dirs = append(dirs, t0Container(svc, cfg)...)
	dirs = append(dirs, t1Container(svc, cfg)...)
	dirs = append(dirs, t3Container(svc, cfg)...)
	return dirs
}

func t0Container(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if svc.Build != nil {
		if !cfg.PodmanVersion.AtLeast(5, 2) {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningFatal,
				Service: svc.Name,
				Field:   "build",
				Message: "requires podman >= 5.2.0",
				Since:   "5.2.0",
			})
		}
		if cfg.PodmanVersion.AtLeast(5, 2) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{svc.Name + ".build"}})
		} else if svc.Image != "" {
			if cfg.PodmanVersion.AtLeast(4, 8) {
				dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{svc.Name + ".image"}})
			} else {
				dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{svc.Image}})
			}
		}
	} else if cfg.PodmanVersion.AtLeast(4, 8) {
		if svc.Image != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{svc.Name + ".image"}})
		}
	} else if svc.Image != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "Image", Values: []string{svc.Image}})
	}
	if svc.ContainerName != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "ContainerName", Values: []string{svc.ContainerName}})
	}
	if svc.Name != "" {
		if cfg.PodmanVersion.AtLeast(5, 3) {
			dirs = append(dirs, c2qtypes.Directive{Key: "ServiceName", Values: []string{svc.Name}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "service_name",
				Message: "requires podman >= 5.3.0",
				Since:   "5.3.0",
			})
		}
	}
	if len(svc.Entrypoint) > 0 {
		if cfg.PodmanVersion.AtLeast(5, 0) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Entrypoint", Values: []string{strings.Join(svc.Entrypoint, " ")}})
		} else {
			dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--entrypoint " + strings.Join(svc.Entrypoint, " ")}})
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningDegraded,
				Service: svc.Name,
				Field:   "entrypoint",
				Message: "using PodmanArgs fallback; upgrade to podman >= 5.0.0 for native Entrypoint= support",
				Since:   "5.0.0",
			})
		}
	}
	if svc.Init != nil && *svc.Init {
		dirs = append(dirs, c2qtypes.Directive{Key: "RunInit", Values: []string{"true"}})
	}
	if svc.ReadOnly {
		dirs = append(dirs, c2qtypes.Directive{Key: "ReadOnly", Values: []string{"true"}})
	}
	if svc.User != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "User", Values: []string{svc.User}})
	}
	if svc.GroupAdd != nil {
		if cfg.PodmanVersion.AtLeast(5, 1) {
			for _, g := range svc.GroupAdd {
				dirs = append(dirs, c2qtypes.Directive{Key: "GroupAdd", Values: []string{g}})
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "group_add",
				Message: "requires podman >= 5.1.0",
				Since:   "5.1.0",
			})
		}
	}
	if svc.WorkingDir != "" {
		if cfg.PodmanVersion.AtLeast(4, 6) {
			dirs = append(dirs, c2qtypes.Directive{Key: "WorkingDir", Values: []string{svc.WorkingDir}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "working_dir",
				Message: "requires podman >= 4.6.0",
				Since:   "4.6.0",
			})
		}
	}

	dirs = append(dirs, SecurityOpts(svc.SecurityOpt, svc.Name, cfg)...)

	if svc.MemLimit > 0 {
		if cfg.PodmanVersion.AtLeast(5, 5) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Memory", Values: []string{strconv.FormatInt(int64(svc.MemLimit), 10)}})
		}
	}
	if svc.Cgroup == "host" {
		if cfg.PodmanVersion.AtLeast(5, 3) {
			dirs = append(dirs, c2qtypes.Directive{Key: "CgroupsMode", Values: []string{"host"}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "cgroup",
				Message: "requires podman >= 5.3.0",
				Since:   "5.3.0",
			})
		}
	}
	if svc.Cgroup == "private" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--cgroupns private"}})
	}

	if svc.UserNSMode != "" {
		if cfg.PodmanVersion.AtLeast(4, 5) {
			dirs = append(dirs, c2qtypes.Directive{Key: "UserNS", Values: []string{svc.UserNSMode}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "userns_mode",
				Message: "requires podman >= 4.5.0",
				Since:   "4.5.0",
			})
		}
	}
	if svc.Hostname != "" {
		if cfg.PodmanVersion.AtLeast(4, 6) {
			dirs = append(dirs, c2qtypes.Directive{Key: "HostName", Values: []string{svc.Hostname}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "hostname",
				Message: "requires podman >= 4.6.0",
				Since:   "4.6.0",
			})
		}
	}
	if svc.ShmSize != 0 {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			dirs = append(dirs, c2qtypes.Directive{Key: "ShmSize", Values: []string{strconv.FormatInt(int64(svc.ShmSize), 10)}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "shm_size",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}
	if len(svc.Sysctls) > 0 {
		if cfg.PodmanVersion.AtLeast(4, 6) {
			sysKeys := sortedKeys(svc.Sysctls)
			for _, k := range sysKeys {
				dirs = append(dirs, c2qtypes.Directive{Key: "Sysctl", Values: []string{k + "=" + svc.Sysctls[k]}})
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "sysctls",
				Message: "requires podman >= 4.6.0",
				Since:   "4.6.0",
			})
		}
	}
	if len(svc.DNS) > 0 {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			for _, d := range svc.DNS {
				dirs = append(dirs, c2qtypes.Directive{Key: "DNS", Values: []string{d}})
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "dns",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}
	if len(svc.DNSSearch) > 0 {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			for _, d := range svc.DNSSearch {
				dirs = append(dirs, c2qtypes.Directive{Key: "DNSSearch", Values: []string{d}})
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "dns_search",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}
	for _, d := range svc.DNSOpts {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			dirs = append(dirs, c2qtypes.Directive{Key: "DNSOption", Values: []string{d}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "dns_opt",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}
	if svc.StopGracePeriod != nil {
		if cfg.PodmanVersion.AtLeast(5, 0) {
			dirs = append(dirs, c2qtypes.Directive{Key: "StopTimeout", Values: []string{time.Duration(*svc.StopGracePeriod).String()}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "stop_grace_period",
				Message: "requires podman >= 5.0.0",
				Since:   "5.0.0",
			})
		}
	}
	if svc.StopSignal != "" {
		if cfg.PodmanVersion.AtLeast(5, 2) {
			dirs = append(dirs, c2qtypes.Directive{Key: "StopSignal", Values: []string{svc.StopSignal}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "stop_signal",
				Message: "requires podman >= 5.2.0",
				Since:   "5.2.0",
			})
		}
	}
	if svc.PullPolicy != "" {
		if cfg.PodmanVersion.AtLeast(4, 6) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Pull", Values: []string{svc.PullPolicy}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "pull_policy",
				Message: "requires podman >= 4.6.0",
				Since:   "4.6.0",
			})
		}
	}

	for _, k := range sortedKeys(svc.Labels) {
		dirs = append(dirs, c2qtypes.Directive{Key: "Label", Values: []string{fmt.Sprintf("%s=%s", k, svc.Labels[k])}})
	}
	for _, k := range sortedKeys(svc.Annotations) {
		dirs = append(dirs, c2qtypes.Directive{Key: "Annotation", Values: []string{fmt.Sprintf("%s=%s", k, svc.Annotations[k])}})
	}
	for _, c := range svc.CapAdd {
		dirs = append(dirs, c2qtypes.Directive{Key: "AddCapability", Values: []string{c}})
	}
	for _, c := range svc.CapDrop {
		dirs = append(dirs, c2qtypes.Directive{Key: "DropCapability", Values: []string{c}})
	}
	for _, e := range svc.Expose {
		dirs = append(dirs, c2qtypes.Directive{Key: "ExposeHostPort", Values: []string{e}})
	}

	for _, dev := range svc.Devices {
		if dev.Target == "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "AddDevice", Values: []string{dev.Source}})
		} else if dev.Permissions != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "AddDevice", Values: []string{dev.Source + ":" + dev.Target + ":" + dev.Permissions}})
		} else {
			dirs = append(dirs, c2qtypes.Directive{Key: "AddDevice", Values: []string{dev.Source + ":" + dev.Target}})
		}
	}

	warnP4Container(svc, cfg)

	return dirs
}

func warnP4Container(svc types.ServiceConfig, cfg *c2qtypes.Config) {
	p4 := func(field, message string) {
		cfg.Warn(c2qtypes.Warning{
			Level:   c2qtypes.WarningSkipped,
			Service: svc.Name,
			Field:   field,
			Message: message,
		})
	}

	if svc.Extends != nil {
		p4("extends", "handled by compose-go loader")
	}
	if len(svc.ExternalLinks) > 0 {
		p4("external_links", "legacy Docker field")
	}
	if len(svc.Links) > 0 {
		p4("links", "legacy Docker field")
	}
	if len(svc.Profiles) > 0 {
		p4("profiles", "handled by comquad at runtime")
	}
	if svc.Scale != nil && *svc.Scale > 0 {
		p4("scale", "replaces deploy.replicas (Swarm orchestration)")
	}
	if svc.DomainName != "" {
		p4("domainname", "legacy Swarm")
	}
	if svc.CredentialSpec != nil {
		p4("credential_spec", "Windows-only")
	}
	if svc.Isolation != "" {
		p4("isolation", "Windows/Swarm")
	}
	if svc.Attach != nil {
		p4("attach", "Docker CLI concept")
	}
	if svc.Develop != nil {
		p4("develop", "dev tooling")
	}
	if len(svc.VolumesFrom) > 0 {
		p4("volumes_from", "Docker-only")
	}
	if svc.CPUCount > 0 {
		p4("cpu_count", "Windows/macOS")
	}
	if svc.CPUPercent > 0 {
		p4("cpu_percent", "Windows/macOS")
	}
	if len(svc.Gpus) > 0 {
		p4("gpus", "no podman equivalent")
	}

	if svc.Deploy != nil {
		if svc.Deploy.Mode != "" {
			p4("deploy.mode", "Swarm orchestration")
		}
		if svc.Deploy.Replicas != nil && *svc.Deploy.Replicas > 0 {
			p4("deploy.replicas", "Swarm orchestration")
		}
		if len(svc.Deploy.Placement.Constraints) > 0 {
			p4("deploy.placement.constraints", "Swarm orchestration")
		}
		if len(svc.Deploy.Placement.Preferences) > 0 {
			p4("deploy.placement.preferences", "Swarm orchestration")
		}
		if svc.Deploy.EndpointMode != "" {
			p4("deploy.endpoint_mode", "Swarm orchestration")
		}
		if len(svc.Deploy.Labels) > 0 {
			p4("deploy.labels", "Swarm orchestration")
		}
		if svc.Deploy.UpdateConfig != nil {
			p4("deploy.update_config", "Swarm orchestration")
		}
		if svc.Deploy.RollbackConfig != nil {
			p4("deploy.rollback_config", "Swarm orchestration")
		}
		if svc.Deploy.Resources.Reservations != nil && len(svc.Deploy.Resources.Reservations.Devices) > 0 {
			p4("deploy.resources.reservations.devices", "Swarm orchestration")
		}
	}

	for _, v := range svc.Volumes {
		if v.Consistency != "" {
			p4("volumes." + v.Target + ".consistency", "inconsequential in podman")
		}
		if v.Volume != nil && v.Volume.Subpath != "" {
			p4("volumes." + v.Target + ".subpath", "Docker engine only (compose 2.27+)")
		}
		if v.Image != nil && v.Image.SubPath != "" {
			p4("volumes." + v.Target + ".subpath", "Docker engine only")
		}
	}

	for name, net := range svc.Networks {
		if net == nil {
			continue
		}
		if net.Priority != 0 {
			p4("networks." + name + ".priority", "Swarm/compose concept")
		}
		if len(net.DriverOpts) > 0 {
			p4("networks." + name + ".driver_opts", "per-service network opts not supported")
		}
	}

	if strings.HasPrefix(svc.Ipc, "service:") {
		p4("ipc:" + svc.Ipc, "no container IPC sharing")
	}
	if strings.HasPrefix(svc.Pid, "service:") {
		p4("pid:" + svc.Pid, "no container PID namespace sharing")
	}
}

func t1Container(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if len(svc.Command) > 0 {
		var quoted []string
		for _, cmd := range svc.Command {
			quoted = append(quoted, "'"+cmd+"'")
		}
		dirs = append(dirs, c2qtypes.Directive{Key: "Exec", Values: []string{strings.Join(quoted, " ")}})
	}

	for _, k := range sortedKeys(svc.Environment) {
		v := svc.Environment[k]
		if v != nil {
			dirs = append(dirs, c2qtypes.Directive{Key: "Environment", Values: []string{k + "=" + *v}})
		} else {
			if cfg.PodmanVersion.AtLeast(5, 6) {
				dirs = append(dirs, c2qtypes.Directive{Key: "Environment", Values: []string{k}})
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: svc.Name,
					Field:   "environment",
					Message: "bare key requires podman >= 5.6.0",
					Since:   "5.6.0",
				})
			}
		}
	}
	for _, f := range svc.EnvFiles {
		dirs = append(dirs, c2qtypes.Directive{Key: "EnvironmentFile", Values: []string{f.Path}})
	}
	for _, t := range svc.Tmpfs {
		dirs = append(dirs, c2qtypes.Directive{Key: "Tmpfs", Values: []string{t}})
	}

	if svc.Logging != nil {
		if svc.Logging.Driver != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "LogDriver", Values: []string{svc.Logging.Driver}})
		}
		if len(svc.Logging.Options) > 0 {
			if cfg.PodmanVersion.AtLeast(5, 2) {
				for _, k := range sortedKeys(svc.Logging.Options) {
					dirs = append(dirs, c2qtypes.Directive{Key: "LogOpt", Values: []string{k + "=" + svc.Logging.Options[k]}})
				}
			} else {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: svc.Name,
					Field:   "logging.options",
					Message: "requires podman >= 5.2.0",
					Since:   "5.2.0",
				})
			}
		}
	} else {
		if svc.LogDriver != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "LogDriver", Values: []string{svc.LogDriver}})
		}
		for _, k := range sortedKeys(svc.LogOpt) {
			if cfg.PodmanVersion.AtLeast(5, 2) {
				dirs = append(dirs, c2qtypes.Directive{Key: "LogOpt", Values: []string{k + "=" + svc.LogOpt[k]}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "logging.options",
				Message: "requires podman >= 5.2.0",
				Since:   "5.2.0",
			})
		}
	}
}

	if svc.PidsLimit != 0 {
		if cfg.PodmanVersion.AtLeast(4, 7) {
			dirs = append(dirs, c2qtypes.Directive{Key: "PidsLimit", Values: []string{strconv.FormatInt(svc.PidsLimit, 10)}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "pids_limit",
				Message: "requires podman >= 4.7.0",
				Since:   "4.7.0",
			})
		}
	}


	if len(svc.ExtraHosts) > 0 {
		if cfg.PodmanVersion.AtLeast(5, 3) {
			for _, host := range sortedKeys(svc.ExtraHosts) {
				for _, ip := range svc.ExtraHosts[host] {
					dirs = append(dirs, c2qtypes.Directive{Key: "AddHost", Values: []string{host + ":" + ip}})
				}
			}
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "extra_hosts",
				Message: "requires podman >= 5.3.0",
				Since:   "5.3.0",
			})
		}
	}

	for _, v := range svc.Volumes {
		if v.Type == types.VolumeTypeBind {
			dirs = append(dirs, c2qtypes.Directive{Key: "Mount", Values: []string{formatBindMount(v, cfg.WorkingDirectory)}})
		} else if v.Type == types.VolumeTypeTmpfs {
			dirs = append(dirs, c2qtypes.Directive{Key: "Tmpfs", Values: []string{formatTmpfsMount(v)}})
		} else {
			vol := v.String()
			if colon := strings.IndexByte(vol, ':'); colon > 0 {
				vol = vol[:colon] + ".volume" + vol[colon:]
			}
			dirs = append(dirs, c2qtypes.Directive{Key: "Volume", Values: []string{vol}})
		}
	}

	for _, p := range svc.Ports {
		dirs = append(dirs, c2qtypes.Directive{Key: "PublishPort", Values: []string{formatPort(p)}})
	}

	if svc.NetworkMode == "host" {
		dirs = append(dirs, c2qtypes.Directive{Key: "Network", Values: []string{"host"}})
	} else if svc.NetworkMode == "none" {
		dirs = append(dirs, c2qtypes.Directive{Key: "Network", Values: []string{"none"}})
	} else if strings.HasPrefix(svc.NetworkMode, "service:") {
		target := strings.TrimPrefix(svc.NetworkMode, "service:")
		if cfg.PodmanVersion.AtLeast(5, 3) {
			dirs = append(dirs, c2qtypes.Directive{Key: "Network", Values: []string{"container:" + target + ".container"}})
		} else {
			cfg.Warn(c2qtypes.Warning{
				Level:   c2qtypes.WarningSkipped,
				Service: svc.Name,
				Field:   "network_mode",
				Message: "network_mode: service:<name> requires podman >= 5.3.0",
				Since:   "5.3.0",
			})
		}
	} else if svc.NetworkMode == "" || svc.NetworkMode == "bridge" {
		for name, net := range svc.Networks {
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
			if len(net.Aliases) > 0 {
				if cfg.PodmanVersion.AtLeast(5, 2) {
					for _, alias := range net.Aliases {
						dirs = append(dirs, c2qtypes.Directive{Key: "NetworkAlias", Values: []string{alias + ":" + name}})
					}
				} else {
					cfg.Warn(c2qtypes.Warning{
						Level:   c2qtypes.WarningSkipped,
						Service: svc.Name,
						Field:   "networks." + name + ".aliases",
						Message: "requires podman >= 5.2.0",
						Since:   "5.2.0",
					})
				}
			}
		}
	}

	return dirs
}

func formatBindMount(v types.ServiceVolumeConfig, workDir string) string {
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
			parts = append(parts, selinuxToRelabel(v.Bind.SELinux))
		}
	}
	return strings.Join(parts, ",")
}

func selinuxToRelabel(selinux string) string {
	if selinux == "Z" {
		return "relabel=private"
	}
	return "relabel=shared"
}

func formatTmpfsMount(v types.ServiceVolumeConfig) string {
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

func t3Container(svc types.ServiceConfig, cfg *c2qtypes.Config) []c2qtypes.Directive {
	var dirs []c2qtypes.Directive

	if svc.Tty {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--tty"}})
	}
	if svc.StdinOpen {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--attach stdin"}})
	}
	if svc.Runtime != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "GlobalArgs", Values: []string{"--runtime " + svc.Runtime}})
	}
	if svc.MacAddress != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--mac-address " + svc.MacAddress}})
	}
	for _, net := range svc.Networks {
		if net != nil && net.MacAddress != "" {
			dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--mac-address " + net.MacAddress}})
		}
	}
	if svc.Ipc == "shareable" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--ipc shareable"}})
	}
	if svc.Pid == "host" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--pid host"}})
	}
	if svc.Uts == "host" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--uts host"}})
	}
	if svc.Privileged {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--privileged"}})
	}
	if svc.MemSwappiness > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{fmt.Sprintf("--memory-swappiness %d", svc.MemSwappiness)}})
	}
	if svc.CPURTRuntime > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{fmt.Sprintf("--cpu-rt-runtime %d", svc.CPURTRuntime)}})
	}
	if svc.CPURTPeriod > 0 {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{fmt.Sprintf("--cpu-rt-period %d", svc.CPURTPeriod)}})
	}
	for _, rule := range svc.DeviceCgroupRules {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--device-cgroup-rule " + rule}})
	}
	for _, k := range sortedKeys(svc.StorageOpt) {
		dirs = append(dirs, c2qtypes.Directive{Key: "GlobalArgs", Values: []string{"--storage-opt " + k + "=" + svc.StorageOpt[k]}})
	}
	if svc.OomKillDisable {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--oom-kill-disable"}})
	}
	if svc.CgroupParent != "" {
		dirs = append(dirs, c2qtypes.Directive{Key: "PodmanArgs", Values: []string{"--cgroup-parent " + svc.CgroupParent}})
	}
	return dirs
}
