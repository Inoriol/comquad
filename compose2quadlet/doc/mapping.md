# Field Mapping Roadmap

Every compose field mapped to quadlet `[Container]`/`[Network]`/`[Volume]`/`[Image]`/`[Build]`
directives, systemd `[Service]`/`[Unit]` directives, or marked unsupported.

**Minimum baseline: Podman 4.4.0** (with pseudo-quadlet fallback for `.image` and `.build`).

Native quadlet type requirements:
- `.container`, `.network`, `.volume`: Podman 4.4.0+
- `.image`: Podman 4.8.0+ (pseudo-quadlet `.cq-image` fallback for 4.4.0-4.7.x)
- `.build`: Podman 5.2.0+ (pseudo-quadlet `.cq-build` fallback for 4.4.0-5.1.x)

See [Pseudo-Quadlet Fallback](#pseudo-quadlet-fallback) section for details.

Sources:
- [Compose Specification](https://github.com/compose-spec/compose-spec/blob/main/spec.md)
- [Quadlet podman-systemd.unit(5)](../podman-systemd.unit.5.md)
- [systemd.resource-control(5)](../systemd.resource-control)
- [systemd.service(5)](../systemd.service)
- [systemd.exec(5)](https://www.freedesktop.org/software/systemd/man/systemd.exec.html)
- [Podman Release Notes](../podman-RELEASE_NOTES.md)

## Version Column Legend

The **Since** column tracks the minimum version required for each mapping:

| Format | Meaning |
|---|---|
| `4.4.0` | Minimum podman version for this quadlet directive |
| `sd 213` | Minimum systemd version for this `[Service]` directive — **reference only** (not used in code; modern podman implies modern systemd) |
| `—` | Not applicable (structural, unsupported, or no specific version constraint) |
| `?` | Version uncertain, needs verification |

### Quadlet Type Introduction

| Type | Podman |
|---|---|
| `.container` | 4.4.0 |
| `.network` | 4.4.0 |
| `.volume` | 4.4.0 |
| `.kube` | 4.4.0 |
| `.image` | 4.8.0 |
| `.pod` | 5.0.0 |
| `.build` | 5.2.0 |
| `.artifact` | 5.7.0 |

## Priority Legend

| P | Meaning |
|---|---------|
| **1** | Direct Quadlet `[Container]`/`[Network]`/`[Volume]`/`[Image]`/`[Build]` directive |
| **2** | Systemd `[Service]` or `[Unit]` directive (cgroup resource control, restart policy, etc.) |
| **3** | PodmanArgs passthrough — no first-class Quadlet directive |
| **4** | Unsupported — orchestration / Swarm / Windows / Docker Desktop concept |
| — | Structural — handled by generating a separate quadlet unit |

### Mapping Principle
**No silent skips.** Every compose field that cannot be mapped at the target podman version must produce a `Warning`. Severity depends on whether the field is skipped (info), degraded to PodmanArgs (warning), or structurally impossible (fatal error).

---

## 1. Image & Runtime

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `image` | 1 | `Image=` | `[Container]` | 4.4.0 | `podman run <image>` |
| `image` (comp. `.image`) | — | — | `[Image]` | 4.8.0 | `podman pull <image>` |
| `build` | — | — | `[Build]` | 5.2.0 | `podman build <context>` |
| `build.context` | — | `SetWorkingDirectory=` | `[Build]` | 5.2.0 | `podman build <context>` (positional) |
| `build.dockerfile` | 1 | `File=` | `[Build]` | 5.2.0 | `podman build -f <file>` |
| `build.args` | 1 | `BuildArg=` | `[Build]` | 5.7.0 | `podman build --build-arg <key>=<value>` |
| `build.target` | 1 | `Target=` | `[Build]` | 5.2.0 | `podman build --target <target>` |
| `build.labels` | 1 | `Label=` | `[Build]` | 5.2.0 | `podman build --label <key>=<value>` |
| `build.network` | 1 | `Network=` | `[Build]` | 5.2.0 | `podman build --network <mode>` |
| `build.no_cache` | 3 | `PodmanArgs=--no-cache` | `[Build]` | 5.2.0 | `podman build --no-cache` |
| `build.secrets` | 1 | `Secret=` | `[Build]` | 5.2.0 | `podman build --secret <spec>` |
| `build.tags` | 1 | `ImageTag=` | `[Build]` | 5.2.0 | `podman build -t <tag>` |
| `build.platforms` | 4 | — | — | — | — |
| `build.extra_hosts` | 4 | — | — | — | — |
| `build.dockerfile_inline` | 4 | — | — | — | — |
| `build.additional_contexts` | 4 | — | — | — | — |
| `build.cache_from` | 4 | — | — | — | — |
| `build.cache_to` | 4 | — | — | — | — |
| `build.entitlements` | 4 | — | — | — | — |
| `build.isolation` | 4 | — | — | — | — |
| `build.privileged` | 4 | — | — | — | — |
| `build.provenance` | 4 | — | — | — | — |
| `build.pull` | 4 | — | — | — | — |
| `build.sbom` | 4 | — | — | — | — |
| `build.ssh` | 4 | — | — | — | — |
| `build.shm_size` | 4 | — | — | — | — |
| `build.ulimits` | 4 | — | — | — | — |
| `command` (string/list) | 1 | `Exec=` | `[Container]` | 4.4.0 | `podman run <image> <command>` |
| `entrypoint` (string/list) | 1 | `Entrypoint=` | `[Container]` | 5.0.0 | `podman run --entrypoint <entrypoint>` |
| `working_dir` | 1 | `WorkingDir=` | `[Container]` | 4.6.0 | `podman run -w <dir>` |
| `user` | 1 | `User=` | `[Container]` | 4.4.0 | `podman run --user <user>` |
| `init` | 1 | `RunInit=` | `[Container]` | 4.4.0 | `podman run --init` |
| `stop_signal` | 1 | `StopSignal=` | `[Container]` | 5.2.0 | `podman run --stop-signal <signal>` |
| `stop_grace_period` | 1 | `StopTimeout=` | `[Container]` | 5.0.0 | `podman run --stop-timeout <seconds>` |
| `tty` | 3 | `PodmanArgs=--tty` | `[Container]` | 4.6.0 | `podman run -t` |
| `stdin_open` | 3 | `PodmanArgs=--attach stdin` | `[Container]` | 4.6.0 | `podman run -a stdin` |
| `pull_policy` | 1 | `Pull=` | `[Container]` | 4.6.0 | `podman run --pull <policy>` |
| `read_only` | 1 | `ReadOnly=` | `[Container]` | 4.4.0 | `podman run --read-only` |
| `runtime` | 3 | `GlobalArgs=--runtime <name>` | `[Container]` | 4.6.0 | `podman --runtime <name> run` |
| `platform` | — | `OS=` / `Arch=` / `Variant=` | `[Image]` | 4.8.0 | `podman pull --os=<> --arch=<> --variant=<>` |
| `post_start` | 4 | — | — | — | — |
| `pre_stop` | 4 | — | — | — | — |
| `pre_start` | 4 | — | — | — | — |
| `models` | 4 | — | — | — | — |
| `provider` | 4 | — | — | — | — |
| `use_api_socket` | 4 | — | — | — | — |
| `domainname` | 4 | — | — | — | — |
| `attach` | 4 | — | — | — | — |
| `develop` | 4 | — | — | — | — |

---

## 2. Networking

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `ports` (short syntax) | 1 | `PublishPort=` | `[Container]` | 4.4.0 | `podman run -p <host>:<container>` |
| `ports` (long syntax) | 1 | `PublishPort=` | `[Container]` | 4.4.0 | `podman run -p <spec>` |
| `ports.app_protocol` | 4 | — | — | — | — |
| `expose` | 1 | `ExposeHostPort=` | `[Container]` | 4.4.0 | `podman run --expose <port>` |
| `networks` | 1 | `Network=` | `[Container]` | 4.4.0 | `podman run --network <name>` |
| `networks.aliases` | 1 | `NetworkAlias=` | `[Container]` | 5.2.0 | `podman run --network-alias <alias>` |
| `networks.ipv4_address` | 1 | `IP=` | `[Container]` | 4.5.0 | `podman run --ip <addr>` |
| `networks.ipv6_address` | 1 | `IP6=` | `[Container]` | 4.5.0 | `podman run --ip6 <addr>` |
| `networks.mac_address` | 3 | `PodmanArgs=--mac-address ...` | `[Container]` | 4.6.0 | `podman run --mac-address <mac>` |
| `networks.priority` | 4 | — | — | — | — |
| `networks.driver_opts` | 4 | — | — | — | — |
| `networks.interface_name` | 4 | — | — | — | — |
| `networks.link_local_ips` | 4 | — | — | — | — |
| `networks.gw_priority` | 4 | — | — | — | — |
| `network_mode: host` | 1 | `Network=host` | `[Container]` | 4.4.0 | `podman run --network host` |
| `network_mode: none` | 1 | `Network=none` | `[Container]` | 4.4.0 | `podman run --network none` |
| `network_mode: service:<name>` | 1 | `Network=container:<name>.container` | `[Container]` | 5.3.0 | `podman run --network container:<name>` |
| `dns` | 1 | `DNS=` | `[Container]` | 4.7.0 | `podman run --dns <server>` |
| `dns_search` | 1 | `DNSSearch=` | `[Container]` | 4.7.0 | `podman run --dns-search <domain>` |
| `dns_opt` | 1 | `DNSOption=` | `[Container]` | 4.7.0 | `podman run --dns-option <option>` |
| `extra_hosts` (list/map) | 1 | `AddHost=` | `[Container]` | 5.3.0 | `podman run --add-host <host:ip>` |
| `hostname` | 1 | `HostName=` | `[Container]` | 4.6.0 | `podman run --hostname <name>` |
| `mac_address` | 3 | `PodmanArgs=--mac-address ...` | `[Container]` | 4.6.0 | `podman run --mac-address <mac>` |
| `ipc: shareable` | 3 | `PodmanArgs=--ipc shareable` | `[Container]` | 4.6.0 | `podman run --ipc shareable` |
| `ipc: service:<name>` | 4 | — | — | — | — |
| `pid: host` | 3 | `PodmanArgs=--pid host` | `[Container]` | 4.6.0 | `podman run --pid host` |
| `pid: service:<name>` | 4 | — | — | — | — |
| `uts: host` | 3 | `PodmanArgs=--uts host` | `[Container]` | 4.6.0 | `podman run --uts host` |

---

## 3. Storage

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `volumes` (short syntax) | 1 | `Volume=` | `[Container]` | 4.4.0 | `podman run -v <src>:<dst>` |
| `volumes` (long, bind) | 1 | `Mount=type=bind,...` | `[Container]` | 4.5.0 | `podman run --mount type=bind,...` |
| `volumes` (long, volume) | 1 | `Volume=` | `[Container]` | 4.4.0 | `podman run -v <vol>:<dst>` |
| `volumes.read_only` | 1 | `Volume=... :ro` | `[Container]` | 4.4.0 | `podman run -v <src>:<dst>:ro` |
| `volumes.selinux` | 1 | `Volume=... :z` / `:Z` | `[Container]` | 4.4.0 | `podman run -v <src>:<dst>:z` |
| `volumes.nocopy` | 1 | `Volume=... :nocopy` | `[Container]` | 4.4.0 | `podman run -v <src>:<dst>:nocopy` |
| `volumes.subpath` | 4 | — | — | — | — |
| `volumes.consistency` | 4 | — | — | — | — |
| `volumes` (long, image) | 4 | — | — | — | — |
| `volumes.image.subpath` | 4 | — | — | — | — |
| `volumes_from` | 4 | — | — | — | — |
| `tmpfs` (string/long) | 1 | `Tmpfs=` | `[Container]` | 4.5.0 | `podman run --tmpfs <spec>` |
| `shm_size` | 1 | `ShmSize=` | `[Container]` | 4.7.0 | `podman run --shm-size <size>` |
| `storage_opt` | 3 | `GlobalArgs=--storage-opt ...` | `[Container]` | 4.6.0 | `podman --storage-opt <opt> run` |

---

## 4. Environment

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `environment` (map) | 1 | `Environment=` | `[Container]` | 4.4.0 | `podman run -e <key>=<value>` |
| `environment` (list) | 1 | `Environment=` | `[Container]` | 4.4.0 | `podman run -e <key>=<value>` |
| `environment` (key-only) | 1 | `Environment=` | `[Container]` | 5.6.0 | `podman run -e <key>` |
| `env_file` (string) | 1 | `EnvironmentFile=` | `[Container]` | 4.4.0 | `podman run --env-file <file>` |
| `env_file` (list) | 1 | `EnvironmentFile=` | `[Container]` | 4.4.0 | `podman run --env-file <file>` |
| `env_file` (`required: false`) | 1 | `EnvironmentFile=` | `[Container]` | 4.4.0 | `podman run --env-file <file>` |
| `env_file` (`format: raw`) | 4 | — | — | — | — |

---

## 5. Security

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `cap_add` | 1 | `AddCapability=` | `[Container]` | 4.4.0 | `podman run --cap-add <cap>` |
| `cap_drop` | 1 | `DropCapability=` | `[Container]` | 4.4.0 | `podman run --cap-drop <cap>` |
| `privileged` | 3 | `PodmanArgs=--privileged` | `[Container]` | 4.6.0 | `podman run --privileged` |
| `security_opt: seccomp=<path>` | 1 | `SeccompProfile=` | `[Container]` | 4.4.0 | `podman run --security-opt seccomp=<path>` |
| `security_opt: apparmor=<profile>` | 1 | `AppArmor=` | `[Container]` | 5.8.0 | `podman run --security-opt apparmor=<profile>` |
| `security_opt: no-new-privileges` | 1 | `NoNewPrivileges=` | `[Container]` | 4.4.0 | `podman run --security-opt no-new-privileges` |
| `security_opt: label=type:<type>` | 1 | `SecurityLabelType=` | `[Container]` | 4.4.0 | `podman run --security-opt label=type:<type>` |
| `security_opt: label=level:<level>` | 1 | `SecurityLabelLevel=` | `[Container]` | 4.4.0 | `podman run --security-opt label=level:<level>` |
| `security_opt: label=filetype:<ft>` | 1 | `SecurityLabelFileType=` | `[Container]` | 4.4.0 | `podman run --security-opt label=filetype:<ft>` |
| `security_opt: label=disable` | 1 | `SecurityLabelDisable=` | `[Container]` | 4.4.0 | `podman run --security-opt label=disable` |
| `security_opt: label=nested` | 1 | `SecurityLabelNested=` | `[Container]` | 4.6.0 | `podman run --security-opt label=nested` |
| `security_opt: mask=<path>` | 1 | `Mask=` | `[Container]` | 4.6.0 | `podman run --security-opt mask=<path>` |
| `security_opt: unmask=<path>` | 1 | `Unmask=` | `[Container]` | 4.6.0 | `podman run --security-opt unmask=<path>` |
| `userns_mode` | 1 | `UserNS=` | `[Container]` | 4.5.0 | `podman run --userns <mode>` |
| `group_add` | 1 | `GroupAdd=` | `[Container]` | 5.1.0 | `podman run --group-add <group>` |
| `secrets` (short syntax) | — | *(handled pre-mapping)* | — | — | `podman run --secret <secret>` |
| `secrets` (long, external) | 1 | `Secret=` | `[Container]` | 4.5.0 | `podman run --secret <secret>` |
| `secrets` (long, file) | 1 | `Volume=<path>:/run/secrets/<name>:ro` | `[Container]` | 4.4.0 | `podman run -v <path>:/run/secrets/<name>:ro` |
| `secrets` (long, environment) | 1 | `Volume=<path>:/run/secrets/<name>:ro` | `[Container]` | 4.4.0 | `podman run -v <path>:/run/secrets/<name>:ro` |
| `configs` (short syntax) | 1 | `Mount=type=bind,source=<path>,target=/<name>` | `[Container]` | 4.5.0 | `podman run --mount type=bind,source=<path>,target=/<name>` |
| `configs` (long syntax) | 1 | `Mount=type=bind,...` | `[Container]` | 4.5.0 | `podman run --mount type=bind,...` |
| `credential_spec` | 4 | — | — | — | — |
| `isolation` | 4 | — | — | — | — |

---

## 6. Resources & Limits

### Memory

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `mem_limit` | 1 | `Memory=` | `[Container]` | 5.5.0 | `podman run -m <size>` |
| `mem_limit` (alt) | 2 | `MemoryMax=` | `[Service]` | sd 231 | `podman run -m <size>` |
| `mem_reservation` | 2 | `MemoryLow=` | `[Service]` | sd 240 | `podman run --memory-reservation <size>` |
| `mem_swappiness` | 3 | `PodmanArgs=--memory-swappiness ...` | `[Container]` | 4.6.0 | `podman run --memory-swappiness <n>` |
| `memswap_limit` | 2 | `MemorySwapMax=` | `[Service]` | sd 232 | `podman run --memory-swap <size>` |

### CPU

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `cpus` | 2 | `CPUQuota=` | `[Service]` | sd 213 | `podman run --cpus <n>` |
| `cpu_shares` | 2 | `CPUWeight=` | `[Service]` | sd 232 | `podman run --cpu-shares <n>` |
| `cpu_period` | 2 | `CPUQuotaPeriodSec=` | `[Service]` | sd 242 | `podman run --cpu-period <period>` |
| `cpu_quota` | 2 | `CPUQuota=` | `[Service]` | sd 213 | `podman run --cpu-quota <quota>` |
| `cpu_rt_runtime` | 3 | `PodmanArgs=--cpu-rt-runtime ...` | `[Container]` | 4.6.0 | `podman run --cpu-rt-runtime <runtime>` |
| `cpu_rt_period` | 3 | `PodmanArgs=--cpu-rt-period ...` | `[Container]` | 4.6.0 | `podman run --cpu-rt-period <period>` |
| `cpu_count` | 4 | — | — | — | — |
| `cpu_percent` | 4 | — | — | — | — |
| `cpuset` | 2 | `AllowedCPUs=` | `[Service]` | sd 244 | `podman run --cpuset-cpus <cpus>` |

### PID / Tasks

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `pids_limit` | 1 | `PidsLimit=` | `[Container]` | 4.7.0 | `podman run --pids-limit <limit>` |
| `pids_limit` (alt) | 2 | `TasksMax=` | `[Service]` | sd 227 | `podman run --pids-limit <limit>` |

### Block I/O (blkio_config)

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `blkio_config.weight` | 2 | `IOWeight=` | `[Service]` | sd 230 | `podman run --blkio-weight <weight>` |
| `blkio_config.weight_device` | 2 | `IODeviceWeight=` | `[Service]` | sd 230 | `podman run --blkio-weight-device <dev>:<weight>` |
| `blkio_config.device_read_bps` | 2 | `IOReadBandwidthMax=` | `[Service]` | sd 230 | `podman run --device-read-bps <dev>:<bps>` |
| `blkio_config.device_write_bps` | 2 | `IOWriteBandwidthMax=` | `[Service]` | sd 230 | `podman run --device-write-bps <dev>:<bps>` |
| `blkio_config.device_read_iops` | 2 | `IOReadIOPSMax=` | `[Service]` | sd 230 | `podman run --device-read-iops <dev>:<iops>` |
| `blkio_config.device_write_iops` | 2 | `IOWriteIOPSMax=` | `[Service]` | sd 230 | `podman run --device-write-iops <dev>:<iops>` |

### OOM

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `oom_score_adj` | 2 | `OOMScoreAdjust=` | `[Service]` | sd 208 | `podman run --oom-score-adj <adj>` |
| `oom_kill_disable` | 3 | `PodmanArgs=--oom-kill-disable` | `[Container]` | 4.6.0 | `podman run --oom-kill-disable` |

### Devices

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `devices` (`HOST:CONTAINER[:PERMS]`) | 1 | `AddDevice=` | `[Container]` | 4.4.0 | `podman run --device <device>` |
| `devices` (CDI syntax) | 1 | `AddDevice=` | `[Container]` | 4.4.0 | `podman run --device <cdi-spec>` |
| `device_cgroup_rules` | 3 | `PodmanArgs=--device-cgroup-rule ...` | `[Container]` | 4.6.0 | `podman run --device-cgroup-rule <rule>` |
| `gpus` | 4 | — | — | — | — |

### Namespace isolation

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `cgroup: host` | 1 | `CgroupsMode=host` | `[Container]` | 5.3.0 | `podman run --cgroups=host` |
| `cgroup: private` | 3 | `PodmanArgs=--cgroupns private` | `[Container]` | 4.6.0 | `podman run --cgroupns private` |
| `cgroup_parent` | 2 | `Slice=` | `[Service]` | sd 208 | `podman run --cgroup-parent <slice>` |
| `cgroup_parent` (alt) | 3 | `PodmanArgs=--cgroup-parent ...` | `[Container]` | 4.6.0 | `podman run --cgroup-parent <path>` |

---

## 7. Healthcheck

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `healthcheck.test` (`CMD`) | 1 | `HealthCmd=` | `[Container]` | 4.5.0 | `podman run --health-cmd <cmd>` |
| `healthcheck.test` (`CMD-SHELL`) | 1 | `HealthCmd=` | `[Container]` | 4.5.0 | `podman run --health-cmd "<cmd>"` |
| `healthcheck.test` (`NONE`) | 1 | *(omit)* | — | 4.5.0 | `podman run --health-cmd none` |
| `healthcheck.disable: true` | 1 | *(omit)* | — | 4.5.0 | `podman run --no-healthcheck` |
| `healthcheck.interval` | 1 | `HealthInterval=` | `[Container]` | 4.5.0 | `podman run --health-interval <interval>` |
| `healthcheck.timeout` | 1 | `HealthTimeout=` | `[Container]` | 4.5.0 | `podman run --health-timeout <timeout>` |
| `healthcheck.retries` | 1 | `HealthRetries=` | `[Container]` | 4.5.0 | `podman run --health-retries <n>` |
| `healthcheck.start_period` | 1 | `HealthStartPeriod=` | `[Container]` | 4.5.0 | `podman run --health-start-period <period>` |
| `healthcheck.start_interval` | 1 | `HealthStartupInterval=` | `[Container]` | 4.5.0 | `podman run --health-startup-interval <interval>` |

---

## 8. Logging

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `logging.driver` | 1 | `LogDriver=` | `[Container]` | 4.5.0 | `podman run --log-driver <driver>` |
| `logging.options` | 1 | `LogOpt=` | `[Container]` | 5.2.0 | `podman run --log-opt <key>=<value>` |

---

## 9. Dependencies & Restart

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `depends_on` (short) | 1 | `After=` / `Requires=` | `[Unit]` | 4.4.0 | — (systemd unit dependency) |
| `depends_on` (long, `condition: service_started`) | 1 | `After=` / `Requires=` | `[Unit]` | 4.4.0 | — (systemd unit dependency) |
| `depends_on` (long, `condition: service_healthy`) | 2 | `ExecStartPre=` (health poll) | `[Service]` | sd 208 | — (systemd health poll) |
| `depends_on` (long, `condition: service_completed_successfully`) | 2 | `After=` / `Requires=` (oneshot dep) | `[Unit]` | 4.4.0 | — (systemd unit dependency) |
| `depends_on` (long, `required: false`) | 1 | `Wants=` (instead of `Requires=`) | `[Unit]` | 4.4.0 | — (systemd unit dependency) |
| `depends_on` (long, `restart: true`) | 2 | `BindsTo=` | `[Unit]` | 4.4.0 | — (systemd unit dependency) |
| `restart: no` | 2 | `Restart=no` | `[Service]` | sd 208 | — (systemd restart policy) |
| `restart: always` | 2 | `Restart=always` | `[Service]` | sd 208 | — (systemd restart policy) |
| `restart: on-failure` | 2 | `Restart=on-failure` | `[Service]` | sd 208 | — (systemd restart policy) |
| `restart: on-failure:<N>` | 2 | `Restart=on-failure` + `StartLimitBurst=N` | `[Service]` | sd 208 | — (systemd restart policy) |
| `restart: unless-stopped` | 2 | `Restart=always` | `[Service]` | sd 208 | — (systemd restart policy) |

---

## 10. Deploy → Systemd `[Service]`

### Resources

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `deploy.resources.limits.cpus` | 2 | `CPUQuota=` | `[Service]` | sd 213 | `podman run --cpus <n>` |
| `deploy.resources.limits.memory` | 2 | `MemoryMax=` | `[Service]` | sd 231 | `podman run -m <size>` |
| `deploy.resources.limits.pids` | 2 | `TasksMax=` | `[Service]` | sd 227 | `podman run --pids-limit <limit>` |
| `deploy.resources.reservations.cpus` | 2 | `CPUWeight=` | `[Service]` | sd 232 | `podman run --cpu-shares <n>` |
| `deploy.resources.reservations.memory` | 2 | `MemoryLow=` | `[Service]` | sd 240 | `podman run --memory-reservation <size>` |
| `deploy.resources.reservations.devices` | 4 | — | — | — | — |

### Restart Policy

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `deploy.restart_policy.condition` | 2 | `Restart=` | `[Service]` | sd 208 | — (systemd restart policy) |
| `deploy.restart_policy.delay` | 2 | `RestartSec=` | `[Service]` | sd 208 | — (systemd restart delay) |
| `deploy.restart_policy.max_attempts` | 2 | `StartLimitBurst=` / `StartLimitIntervalSec=` | `[Service]` | sd 208 | — (systemd rate limiting) |
| `deploy.restart_policy.window` | 2 | `RuntimeMaxSec=` | `[Service]` | sd 229 | — (systemd runtime limit) |

### Orchestration (unsupported for single-node)

| Compose field | P | Since | Command line equivalent |
|---|---|---|---|
| `deploy.mode` | 4 | — | — |
| `deploy.replicas` | 4 | — | — |
| `deploy.placement.constraints` | 4 | — | — |
| `deploy.placement.preferences` | 4 | — | — |
| `deploy.endpoint_mode` | 4 | — | — |
| `deploy.labels` | 4 | — | — |
| `deploy.update_config` | 4 | — | — |
| `deploy.rollback_config` | 4 | — | — |

---

## 11. Image → `.image` Quadlet

Generated as companion unit for every service with `image:`.

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `image` | 1 | `Image=` | `[Image]` | 4.8.0 | `podman pull <image>` |
| `pull_policy` | 1 | `Policy=` | `[Image]` | 5.6.0 | `podman pull --policy <policy>` |
| `platform` (OS) | 1 | `OS=` | `[Image]` | 4.8.0 | `podman pull --os=<os>` |
| `platform` (arch) | 1 | `Arch=` | `[Image]` | 4.8.0 | `podman pull --arch=<arch>` |
| `platform` (variant) | 1 | `Variant=` | `[Image]` | 4.8.0 | `podman pull --variant=<variant>` |
| image pull retries | 1 | `Retry=` | `[Image]` | 5.5.0 | `podman pull --retry=<n>` |
| image pull retry delay | 1 | `RetryDelay=` | `[Image]` | 5.5.0 | `podman pull --retry-delay=<delay>` |

---

## 12. Network Top-Level → `.network` Quadlet

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `networks.<name>.driver` | 1 | `Driver=` | `[Network]` | 4.4.0 | `podman network create --driver <driver>` |
| `networks.<name>.driver_opts` | 1 | `Options=` | `[Network]` | 4.4.0 | `podman network create --opt <key>=<value>` |
| `networks.<name>.ipam.driver` | 1 | `IPAMDriver=` | `[Network]` | 4.4.0 | `podman network create --ipam-driver <driver>` |
| `networks.<name>.ipam.config[].subnet` | 1 | `Subnet=` | `[Network]` | 4.4.0 | `podman network create --subnet <subnet>` |
| `networks.<name>.ipam.config[].gateway` | 1 | `Gateway=` | `[Network]` | 4.4.0 | `podman network create --gateway <gateway>` |
| `networks.<name>.ipam.config[].ip_range` | 1 | `IPRange=` | `[Network]` | 4.4.0 | `podman network create --ip-range <range>` |
| `networks.<name>.ipam.config[].aux_addresses` | 4 | — | — | — | — |
| `networks.<name>.internal` | 1 | `Internal=` | `[Network]` | 4.4.0 | `podman network create --internal` |
| `networks.<name>.enable_ipv6` | 1 | `IPv6=` | `[Network]` | 4.4.0 | `podman network create --ipv6` |
| `networks.<name>.external` | — | *(skip generation)* | — | — | — |
| `networks.<name>.labels` | 1 | `Label=` | `[Network]` | 5.6.0 | `podman network create --label <key>=<value>` |
| `networks.<name>.attachable` | 4 | — | — | — | — |
| `networks.<name>.dns` | 1 | `DNS=` | `[Network]` | 4.7.0 | `podman network create --dns <server>` |
| `networks.<name>.interface_name` | 1 | `InterfaceName=` | `[Network]` | 5.6.0 | `podman network create --interface-name <name>` |
| `networks.<name>.disable_dns` | 1 | `DisableDNS=` | `[Network]` | 4.4.0 | `podman network create --disable-dns` |
| `networks.<name>.delete_on_stop` | 1 | `NetworkDeleteOnStop=` | `[Network]` | 5.5.0 | — (quadlet lifecycle management) |
| `networks.<name>.name` | 1 | `NetworkName=` | `[Network]` | 4.7.0 | `podman network create <name>` (positional) |

---

## 13. Volume Top-Level → `.volume` Quadlet

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `volumes.<name>.driver` | 1 | `Driver=` | `[Volume]` | 4.7.0 | `podman volume create --driver <driver>` |
| `volumes.<name>.driver_opts` | 1 | `Options=` | `[Volume]` | 6.0.0 | `podman volume create --opt <key>=<value>` |
| `volumes.<name>.external` | — | *(skip generation)* | — | — | — |
| `volumes.<name>.labels` | 1 | `Label=` | `[Volume]` | 4.7.0 | `podman volume create --label <key>=<value>` |
| `volumes.<name>.name` | 1 | `VolumeName=` | `[Volume]` | 4.7.0 | `podman volume create <name>` (positional) |
| `volumes.<name>.uid` | 1 | `UID=` | `[Volume]` | 6.0.0 | `podman volume create --uid <uid>` |
| `volumes.<name>.gid` | 1 | `GID=` | `[Volume]` | 6.0.0 | `podman volume create --gid <gid>` |
| `volumes.<name>.copy` | 1 | `Copy=` | `[Volume]` | 4.7.0 | `podman volume create --opt o=copy` |
| `volumes.<name>.device` | 1 | `Device=` | `[Volume]` | 4.7.0 | `podman volume create --opt device=<device>` |
| `volumes.<name>.type` | 1 | `Type=` | `[Volume]` | 4.7.0 | `podman volume create --opt type=<type>` |

---

## 14. Metadata & Sysctl

| Compose field | P | Target directive | Target section | Since | Command line equivalent |
|---|---|---|---|---|---|
| `labels` (map/list) | 1 | `Label=` | `[Container]` | 4.4.0 | `podman run --label <key>=<value>` |
| `label_file` | 1 | `Label=` | `[Container]` | — | `podman run --label-file <file>` |
| `annotations` (map/list) | 1 | `Annotation=` | `[Container]` | 4.4.0 | `podman run --annotation <key>=<value>` |
| `container_name` | 1 | `ContainerName=` | `[Container]` | 4.4.0 | `podman run --name <name>` |
| `service_name` (derived from svc.Name) | 1 | `ServiceName=` | `[Container]` | 5.3.0 | — (quadlet unit naming) |
| `sysctls` (map/list) | 1 | `Sysctl=` | `[Container]` | 4.6.0 | `podman run --sysctl <key>=<value>` |
| `ulimits` | 2 | `Limit*= ` | `[Service]` | sd 208 | `podman run --ulimit <type>=<soft>:<hard>` |
| `ulimits` (alt) | 1 | `Ulimit=` | `[Container]` | 4.7.0 | `podman run --ulimit <type>=<soft>:<hard>` |

---

## 15. Completely Unsupported

| Field | Reason |
|---|---|
| `extends` | Multi-file composition; handled by compose-go loader at parse time |
| `external_links` | Legacy Docker feature; not a podman concept |
| `links` | Legacy Docker feature; use `depends_on` + `networks` |
| `profiles` | Runtime selection; comquad handles at orchestration level |
| `scale` | Orchestration concept (replaces `deploy.replicas`) |
| `domainname` | Swarm legacy |
| `credential_spec` | Windows-only |
| `isolation` | Windows/Swarm |

---

## 16. Quadlet-Only Directives (No Compose Equivalent)

These directives are supported by Quadlet but have **no corresponding Compose spec field**.
They are available via the **x-extension mechanism** (see `x-container`, `x-image`, `x-build`, `x-network`, `x-volume`, `x-systemd` in README.md).

### 16.1 `[Container]` — Quadlet-only (via `x-container`)

| Directive | Podman equivalent | Since | Notes |
|---|---|---|---|
| `ContainersConfModule=` | `--module=` | 4.7.0 | Load containers.conf(5) module |
| `EnvironmentHost=` | `--env-host` | 4.6.0 | Pass host environment into container |
| `GIDMap=` | `--gidmap=` | 4.8.0 | GID mapping for user namespace |
| `Group=` | `--user UID:GID` | 4.4.0 | GID to run as (complement to `User=`) |
| `HealthLogDestination=` | `--health-log-destination=` | 5.5.0 | Health check log destination |
| `HealthMaxLogCount=` | `--health-max-log-count=` | 5.5.0 | Max health check log rotations |
| `HealthMaxLogSize=` | `--health-max-log-size=` | 5.5.0 | Max health check log size |
| `HealthOnFailure=` | `--health-on-failure=` | 5.0.0 | Action when container becomes unhealthy |
| `HealthStartupCmd=` | `--health-startup-cmd=` | 5.0.0 | Startup health check command |
| `HealthStartupInterval=` | `--health-startup-interval=` | 5.0.0 | Startup health check interval |
| `HealthStartupRetries=` | `--health-startup-retries=` | 5.0.0 | Startup health check retries |
| `HealthStartupSuccess=` | `--health-startup-success=` | 5.0.0 | Required successful startup checks |
| `HealthStartupTimeout=` | `--health-startup-timeout=` | 5.0.0 | Startup health check timeout |
| `HttpProxy=` | `--http-proxy=` | 5.7.0 | Control proxy env var forwarding |
| `ImageVolume=` | `--image-volume=` | 6.1.0 | How to handle builtin image volumes |
| `Notify=` | `--sdnotify=` | 5.0.0 | sd_notify support (also `healthy`) |
| `Pod=` | `--pod=` | 5.0.0 | Link container to a `.pod` quadlet |
| `ReadOnlyTmpfs=` | `--read-only-tmpfs` | 4.8.0 | Mount tmpfs on read-only containers |
| `ReloadCmd=` | *(generates ExecReload)* | 5.5.0 | Command for `systemctl reload` via exec |
| `ReloadSignal=` | *(generates ExecReload)* | 5.5.0 | Signal for `systemctl reload` via kill |
| `Retry=` | `--retry=` | 5.5.0 | Image pull retry count |
| `RetryDelay=` | `--retry-delay=` | 5.5.0 | Delay between image pull retries |
| `Rootfs=` | `--rootfs=` | 4.5.0 | Root filesystem path (alternative to Image) |
| `StartWithPod=` | *(implicit)* | 5.0.0 | Start container when pod starts |
| `SubGIDMap=` | `--subgidname=` | 4.8.0 | Subordinate GID map by name |
| `SubUIDMap=` | `--subuidname=` | 4.8.0 | Subordinate UID map by name |
| `Timezone=` | `--tz=` | 4.6.0 | Container timezone |
| `UIDMap=` | `--uidmap=` | 4.8.0 | UID mapping for user namespace |
| `Umask=` | `--umask=` | 4.8.0 | Container process umask |

### 16.2 `[Network]` — Quadlet-only (via `x-network`)

| Directive | Podman equivalent | Since | Notes |
|---|---|---|---|
| `ContainersConfModule=` | `--module=` | 4.7.0 | Load containers.conf(5) module |

### 16.3 `[Volume]` — Quadlet-only (via `x-volume`)

| Directive | Podman equivalent | Since | Notes |
|---|---|---|---|
| `ContainersConfModule=` | `--module=` | 4.7.0 | Load containers.conf(5) module |
| `Group=` | *(chown)* | 6.0.0 | GID owner of volume mount point |
| `Image=` | *(artifact volume)* | 6.0.0 | Back volume with OCI artifact image |
| `User=` | *(chown)* | 6.0.0 | UID owner of volume mount point |

### 16.4 `[Build]` — Quadlet-only (via `x-build`)

| Directive | Podman equivalent | Since | Notes |
|---|---|---|---|
| `Annotation=` | `--annotation=` | 5.2.0 | OCI annotations on built image |
| `Arch=` | `--arch=` | 5.2.0 | Target architecture |
| `AuthFile=` | `--authfile=` | 5.2.0 | Registry authentication file |
| `ContainersConfModule=` | `--module=` | 5.2.0 | Load containers.conf(5) module |
| `DNS=` | `--dns=` | 5.2.0 | DNS servers during build |
| `DNSOption=` | `--dns-option=` | 5.2.0 | DNS options during build |
| `DNSSearch=` | `--dns-search=` | 5.2.0 | DNS search domains during build |
| `Environment=` | `--env=` | 5.2.0 | Environment variables during build |
| `ForceRM=` | `--force-rm=` | 5.2.0 | Always remove intermediate containers |
| `Pull=` | `--pull=` | 5.2.0 | Pull policy during build |
| `Retry=` | `--retry=` | 5.5.0 | Build retry count |
| `RetryDelay=` | `--retry-delay=` | 5.5.0 | Delay between build retries |
| `TLSVerify=` | `--tls-verify=` | 5.2.0 | Require TLS verification for registries |
| `Variant=` | `--variant=` | 5.2.0 | Target platform variant |
| `Volume=` | `--volume=` | 5.2.0 | Mount volume during build |

### 16.5 `[Image]` — Quadlet-only (via `x-image`)

| Directive | Podman equivalent | Since | Notes |
|---|---|---|---|
| `AllTags=` | `--all-tags` | 4.8.0 | Pull all tagged images |
| `AuthFile=` | `--authfile=` | 4.8.0 | Registry authentication file |
| `CertDir=` | `--cert-dir=` | 4.8.0 | TLS certificates directory |
| `ContainersConfModule=` | `--module=` | 4.8.0 | Load containers.conf(5) module |
| `Creds=` | `--creds=` | 4.8.0 | Registry credentials |
| `DecryptionKey=` | `--decryption-key=` | 4.8.0 | Key for decrypting encrypted image |
| `ImageTag=` | `--tag=` | 5.3.0 | Additional tag for pulled image |
| `TLSVerify=` | `--tls-verify=` | 4.8.0 | Require TLS verification for registries |

### 16.6 `.pod` Quadlet (No Compose Equivalent)

Pod quadlets have no Compose equivalent at all. The entire `.pod` unit type is Quadlet-specific.

| Directive | Podman equivalent | Since | Notes |
|---|---|---|---|
| `AddHost=` | `--add-host=` | 5.0.0 | Host-to-IP mapping for pod |
| `ContainersConfModule=` | `--module=` | 5.0.0 | Load containers.conf(5) module |
| `DNS=` | `--dns=` | 5.0.0 | DNS servers for pod |
| `DNSOption=` | `--dns-option=` | 5.0.0 | DNS options for pod |
| `DNSSearch=` | `--dns-search=` | 5.0.0 | DNS search domains for pod |
| `ExitPolicy=` | `--exit-policy=` | 5.6.0 | Pod exit behavior (`stop`/`continue`) |
| `GIDMap=` | `--gidmap=` | 5.0.0 | GID mapping for pod namespace |
| `GlobalArgs=` | *(global podman args)* | 5.0.0 | Arguments between podman and pod |
| `HostName=` | `--hostname=` | 5.0.0 | Hostname for pod |
| `IP=` | `--ip=` | 5.0.0 | Static IPv4 for pod |
| `IP6=` | `--ip6=` | 5.0.0 | Static IPv6 for pod |
| `Label=` | `--label=` | 5.0.0 | OCI labels on pod |
| `Network=` | `--network=` | 5.0.0 | Network for pod |
| `NetworkAlias=` | `--network-alias=` | 5.0.0 | Network-scoped alias for pod |
| `PodmanArgs=` | *(pod create args)* | 5.0.0 | Additional pod create arguments |
| `PodName=` | `--name=` | 5.0.0 | Custom pod name |
| `PublishPort=` | `--publish=` | 5.0.0 | Port publishing for pod |
| `ServiceName=` | *(systemd unit name)* | 5.0.0 | Systemd service name |
| `ShmSize=` | `--shm-size=` | 5.0.0 | Shared memory size for pod |
| `StopTimeout=` | `--time=` | 5.7.0 | Stop timeout for pod |
| `SubGIDMap=` | `--subgidname=` | 5.0.0 | Subordinate GID map by name |
| `SubUIDMap=` | `--subuidname=` | 5.0.0 | Subordinate UID map by name |
| `UIDMap=` | `--uidmap=` | 5.0.0 | UID mapping for pod namespace |
| `UserNS=` | `--userns=` | 5.0.0 | User namespace mode for pod |
| `Volume=` | `--volume=` | 5.0.0 | Volume mount for pod infra container |

### 16.7 `.kube` Quadlet (No Compose Equivalent)

Kube quadlets deploy from Kubernetes YAML. Entirely Quadlet-specific.

| Directive | Podman equivalent | Since | Notes |
|---|---|---|---|
| `AutoUpdate=` | `--label io.containers.autoupdate=` | 4.4.0 | Auto-update policy |
| `ConfigMap=` | `--configmap=` | 4.4.0 | ConfigMap YAML path |
| `ContainersConfModule=` | `--module=` | 4.7.0 | Load containers.conf(5) module |
| `ExitCodePropagation=` | `--exit-code-propagation=` | 4.4.0 | How to propagate exit codes |
| `GlobalArgs=` | *(global podman args)* | 4.4.0 | Arguments between podman and kube |
| `KubeDownForce=` | `--force=` | 4.5.0 | Force removal on kube down |
| `LogDriver=` | `--log-driver=` | 4.4.0 | Log driver for kube containers |
| `Network=` | `--network=` | 4.4.0 | Network for kube containers |
| `PodmanArgs=` | *(kube play args)* | 4.4.0 | Additional kube play arguments |
| `PublishPort=` | `--publish=` | 4.4.0 | Port publishing |
| `ServiceName=` | *(systemd unit name)* | 4.4.0 | Systemd service name |
| `SetWorkingDirectory=` | `--context-dir=` | 4.4.0 | Working directory for kube |
| `UserNS=` | `--userns=` | 4.5.0 | User namespace mode |
| `Yaml=` | *(positional arg)* | 4.4.0 | Path to Kubernetes YAML |

### 16.8 `.artifact` Quadlet (No Compose Equivalent)

Artifact quadlets manage OCI artifacts. Entirely Quadlet-specific (since 5.7.0).

| Directive | Podman equivalent | Since | Notes |
|---|---|---|---|
| `Artifact=` | *(positional arg)* | 5.7.0 | Artifact identifier |
| `AuthFile=` | `--authfile=` | 5.7.0 | Registry authentication file |
| `CertDir=` | `--cert-dir=` | 5.7.0 | TLS certificates directory |
| `ContainersConfModule=` | `--module=` | 5.7.0 | Load containers.conf(5) module |
| `Creds=` | `--creds=` | 5.7.0 | Registry credentials |
| `DecryptionKey=` | `--decryption-key=` | 5.7.0 | Key for decrypting encrypted artifact |
| `GlobalArgs=` | *(global podman args)* | 5.7.0 | Arguments between podman and artifact |
| `PodmanArgs=` | *(artifact pull args)* | 5.7.0 | Additional artifact pull arguments |
| `Quiet=` | `--quiet` | 5.7.0 | Suppress output |
| `Retry=` | `--retry=` | 5.7.0 | Pull retry count |
| `RetryDelay=` | `--retry-delay=` | 5.7.0 | Delay between pull retries |
| `ServiceName=` | *(systemd unit name)* | 5.7.0 | Systemd service name |
| `TLSVerify=` | `--tls-verify=` | 5.7.0 | Require TLS verification |

### 16.9 `[Quadlet]` Section

| Directive | Since | Notes |
|---|---|---|
| `DefaultDependencies=` | 4.7.0 | Disable implicit network-online.target dependencies |

---

## 17. Mapping Counts

| Priority | Count | Min podman/systemd |
|---|---|---|
| **1** Direct Quadlet | ~77 fields | 4.4.0 baseline, ~15 later additions |
| **2** Systemd `[Service]`/`[Unit]` | ~30 fields | sd 208 baseline, ~5 later additions |
| **3** PodmanArgs passthrough | ~15 fields | 4.6.0 (PodmanArgs added) |
| **4** Unsupported | ~52 fields | — |
| **—** Structural | ~10 fields | 4.4.0 baseline |

**Total: ~184 compose fields tracked.**

### Quadlet-only directives (no Compose equivalent)

| Section | Count | Notes |
|---|---|---|
| `[Container]` | 29 | User namespaces, health startup, reload, timezone, … |
| `[Network]` | 1 | `ContainersConfModule=` |
| `[Volume]` | 4 | `User=`/`Group=`/`Image=`/`ContainersConfModule=` |
| `[Build]` | 15 | Registry auth, DNS, env, retry, … |
| `[Image]` | 8 | Registry auth, TLS, all-tags, … |
| `.pod` | 25 | Entire pod unit type |
| `.kube` | 14 | Entire kube unit type |
| `.artifact` | 13 | Entire artifact unit type |
| `[Quadlet]` | 1 | `DefaultDependencies=` |

**Total: ~110 quadlet-only directives**

## Version Requirement Summary

### Must have (baseline: podman 4.4.0)
All core [Container], [Network], [Volume] quadlet types.
~70% of priority-1 fields available.

### Pseudo-quadlet fallback (4.4.0+)
- `.cq-image` → `.service` with `ExecStart=podman pull` for Podman 4.4.0-4.7.x
- `.cq-build` → `.service` with `ExecStart=podman build` for Podman 4.4.0-5.1.x

### Native quadlet types
- 4.8.0: `.image` type (native image pull management)
- 5.0.0: `Entrypoint=`, `StopTimeout=`, `Notify=`, `Pod=`, `.pod` type
- 5.2.0: `.build` type, `NetworkAlias=`, `StopSignal=`, `LogOpt=`
- 5.3.0: `CgroupsMode=`, `AddHost=`, `ServiceName=`, `StartWithPod=`
- 5.5.0: `Memory=`, `Retry=`/`RetryDelay=`, `ReloadCmd=`/`ReloadSignal=`
- 5.6.0: `Policy=`, `InterfaceName=`, env key-only, `ExitPolicy=` (pod)
- 5.7.0: `BuildArg=` (build), `HttpProxy=`, `IgnoreFile=`, `.artifact` type
- 5.8.0: `AppArmor=`
- 6.0.0: Volume `UID=`/`GID=`/`Options=`/`User=`/`Group=`/`Image=`
- 6.1.0: `ImageVolume=`

---

## Pseudo-Quadlet Fallback

When targeting Podman versions below the native quadlet type requirements, comquad generates
pseudo-quadlet units (`.cq-image`, `.cq-build`) that are converted to standard `.service` units
by the pseudo-quadlet generator.

### `.cq-image` Fallback (Podman 4.4.0-4.7.x)

For services with `image:` when Podman < 4.8.0:

**Input:** Service with `image: nginx:latest`

**Generated `.cq-image` unit:**
```ini
[Image]
Image=docker.io/library/nginx:latest
```

**Converted `.service` unit:**
```ini
[Unit]
Description=comquad image pull: nginx

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/podman pull docker.io/library/nginx:latest

[Install]
WantedBy=default.target
```

Container units reference the service via `After=<name>.service` instead of `Image=<name>.image`.

### `.cq-build` Fallback (Podman 4.4.0-5.1.x)

For services with `build:` when Podman < 5.2.0:

**Input:** Service with `build: ./app`

**Generated `.cq-build` unit:**
```ini
[Build]
SetWorkingDirectory=./app
ImageTag=myapp_app:latest
```

**Converted `.service` unit:**
```ini
[Unit]
Description=comquad image build: app

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/path/to/app
ExecStart=/usr/bin/podman build -t myapp_app:latest .

[Install]
WantedBy=default.target
```

### Disabling Fallback

Use `WithoutPseudoQuadlet()` to disable the fallback. When disabled:
- `.image` units are not generated for Podman < 4.8.0 (images pulled via container unit)
- `.build` units cause a fatal error for Podman < 5.2.0 (original behavior)

### Limitations of Pseudo-Quadlet

- No native quadlet dependency tracking (uses systemd `After=` instead)
- No `Policy=` support for image pulls (always pulls)
- No `Retry=`/`RetryDelay=` support (systemd handles restarts)
- Build args (`build.args`) not supported in fallback mode (requires Podman 5.7.0+)
- Platform-specific pulls not supported in fallback mode
