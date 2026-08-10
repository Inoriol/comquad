## 🗺️ Roadmap & Next Steps

For long term goals refer to [Roadmap](./ROADMAP.md).

---

## 🧩 Missing Features

### Build Processing ✅ (Implemented)

Build blocks are now fully supported. The implementation bypasses podlet's fragile build support entirely:

1. ~~**Anonymous builds**~~ — Image tags (`<project>-<service>:latest`) are injected for services with `build:` and no explicit `image:`.
2. ~~**Path fixing**~~ — Build contexts and Dockerfile paths are resolved to absolute paths in the preprocessor.
3. ~~**Dockerfile patching**~~ — `FROM` lines are patched with `docker.io/library/` prefixes for consistent image resolution. Stage aliases are tracked and preserved. Patched Dockerfiles are written to `$XDG_CACHE_HOME/comquad/builds/<project>/`.
4. ~~**Build image detection**~~ — `handleImages()` detects containers with matching `.build` files and skips registry pulls. `ImageQuadletHandler` skips `.image` generation for built containers.
5. ~~**`.build` quadlet generation**~~ — `BuildQuadletHandler` in the graft step generates `.build` quadlet files directly. `.container` files reference `.build` files via `Image=cq-<project>-<service>.build`, creating proper systemd dependency chains. `AutoUpdate=registry` is stripped from built containers.
6. **Containerfile support** — `$XDG_CACHE_HOME/comquad/builds/` is cleaned up on `comquad down`. `Containerfile` is checked before `Dockerfile` as default build file.

### Deploy

The compose `deploy:` section is fully rejected by podlet (level 0). comquad must intercept and strip the entire `deploy:` block during preprocessing, then translate its sub-fields into native systemd resource-control directives and quadlet options via the graft step.

**Resource Limits & Reservations**

Map `deploy.resources` to systemd cgroup resource-control directives injected in `[Service]`. These provide stronger guarantees than podman CLI flags because they are enforced at the kernel cgroup level and apply to the entire unit's process tree. Quadlet passes unknown `[Service]` directives through to the generated `.service` file.

| Compose field | systemd directive | Example |
|---|---|---|
| `resources.limits.cpus` | `CPUQuota=` | `cpus: '0.50'` → `CPUQuota=50%` |
| `resources.limits.memory` | `MemoryMax=` | `memory: 50M` → `MemoryMax=50M` |
| `resources.limits.pids` | `TasksMax=` | `pids: 1` → `TasksMax=1` |
| `resources.reservations.cpus` | `CPUWeight=` | `cpus: '0.25'` → proportional weight |
| `resources.reservations.memory` | `MemoryLow=` | `memory: 20M` → best-effort protection |
| `resources.reservations.devices` | `AddDevice=` | Leverages podlet's level-2 device support |

If both `deploy.resources.limits.memory` and `mem_limit` (which podlet maps to `Memory=` at level 2) are present, the deploy handler should take precedence and strip the native field to avoid conflicting resource constraints.

**Restart Policy**

Map `deploy.restart_policy` to systemd service directives. When `deploy.restart_policy` is present, it overrides the top-level `restart:` field per the compose spec. The handler must strip `restart:` to avoid conflicts.

| Compose field | systemd directive(s) | Notes |
|---|---|---|
| `restart_policy.condition: none` | `Restart=no` | |
| `restart_policy.condition: on-failure` | `Restart=on-failure` | |
| `restart_policy.condition: any` | `Restart=always` | Default compose behavior |
| `restart_policy.delay` | `RestartSec=` | Duration between restart attempts |
| `restart_policy.max_attempts` | `StartLimitBurst=` | Combined with `StartLimitIntervalSec=` for the time window |
| `restart_policy.window` | `RuntimeMaxSec=` | Approximate: kills the service if it hasn't stabilized within the window. Docker Swarm uses the window to decide whether a restart counts toward `max_attempts`; systemd has no equivalent fine-grained success window. |

Systemd provides richer restart behavior than the compose spec via `RestartSteps=` and `RestartMaxDelaySec=` (exponential backoff), `RestartMode=direct` (skip failed state on restart), and `RestartForceExitStatus=` / `RestartPreventExitStatus=`. These can be exposed through `x-comquad-` extensions.

**Mode & Replicas**

- `deploy.mode: global` — Not applicable. Quadlets are single-instance by design. Multiple nodes would require external orchestration.
- `deploy.mode: replicated` + `deploy.replicas` — Not directly mappable. Quadlets create exactly one unit per service. Multi-instance use cases should use template units (`servicename@.container`) or a higher-level orchestrator.

**Endpoint Mode**

- `deploy.endpoint_mode: vip` — Default behavior with `NetworkAlias=` already injected by the cooker.
- `deploy.endpoint_mode: dnsrr` — Achievable with `NetworkAlias=` on multiple containers, but limited applicability without replica support.

**Placement, Update Config & Rollback Config**

`deploy.placement` (constraints, preferences), `deploy.update_config`, and `deploy.rollback_config` are swarm orchestration features that require a multi-node cluster. Not applicable to single-instance quadlet deployments. Log warnings when encountered.

**Implementation Strategy**

1. **Preprocess**: Strip the entire `deploy:` block from the compose YAML before podlet sees it.
2. **Graft**: A `DeployHandler` (parallel to `SecretHandler` and `ImageQuadletHandler`) processes each service's stripped deploy spec and injects the corresponding `[Service]` directives into quadlet files.
3. **Warnings**: Log warnings for unsupported fields (`mode`, `replicas`, `placement`, `update_config`, `rollback_config`, `endpoint_mode`).

### Service-field Grafting Opportunities

The graft step can inject arbitrary systemd `[Service]` directives beyond deploy-specific ones. The following compose fields have no direct podlet equivalent but can be covered through grafting:

1. **`env_file:`** — Parse compose `env_file:` blocks and write them to managed files, then inject `EnvironmentFile=` in `[Service]`. Podlet already supports `EnvironmentFile=` at level 2, but offloading resolution to comquad allows preprocessing (e.g. `.env` fallback, variable interpolation) before passing a clean file to quadlet.

2. **`depends_on:` conditions** — Podlet rejects `depends_on.condition` (level 0: `service_healthy`, `service_completed_successfully`, `service_started`). These can be mapped to `ExecStartPre=` commands that poll dependent services via `systemctl is-active` or `podman healthcheck run` before the container starts. The simpler `depends_on` (without conditions) is already handled by podlet at level 2 (→ `Requires=`, `After=`).

3. **`stop_grace_period`** — Already handled by podlet at level 2 (→ `StopTimeout=`). Can also map to systemd `TimeoutStopSec=` for an additional safety layer.

### Other Gaps

- **`configs:` compose section** — Similar to secrets but for non-sensitive configuration files. Could use the same direct bind mount pattern.
- **Long syntax for secrets** — `target:` in service-level secret references (for custom mount paths) is partially supported via `SecretRef.Target`. Full implementation needs end-to-end testing.
- **Swarm mode compatibility** — Swarm-specific compose extensions (deploy modes, constraints, replicas) are not supported.

### Future Security Improvements

- **Tmpfs-backed secrets via `LoadCredential`** — Currently secrets are bind-mounted directly from managed files on disk. Consider generating a companion `.service` service unit file (not a `.container` quadlet) that uses systemd `LoadCredential=` in `[Service]` combined with `Volume=%d/<name>` to mount secrets from systemd's RAM-backed credential directories (`/run/credentials/`). This would keep secret values in tmpfs memory rather than on persistent storage. Initial implementation attempted this using quadlet's `[Service]` pass-through, but `LoadCredential=` + `Volume=` with credential paths didn't integrate correctly with quadlet's container lifecycle. A standalone `.service` file could bypass quadlet entirely for credential setup.

---

## 🗺️ Long-Term Roadmap Goals

For reference, these are tracked in [ROADMAP.md](./ROADMAP.md).
