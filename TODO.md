## 🗺️ Roadmap & Next Steps

For long term goals refer to [Roadmap](./ROADMAP.md).

---

## 🧩 Missing Features

### Build Processing

Currently, `build:` blocks in compose files are explicitly rejected. Build support needs to:

1. **Anonymous builds** — Adjust build contexts and Dockerfile paths for podlet. Podlet expects `build:` context paths to be absolute or properly relative to the compose file directory. The preprocessor should normalize these paths (similar to volume path resolution).

2. **Dockerfile patching** — Inject explicit `docker.io/` registry prefixes into `FROM` lines of user-provided Dockerfiles to ensure consistent image resolution. Patched Dockerfiles should be written to `$XDG_CACHE_HOME/comquad/builds/<project>/` and referenced from the generated `.build` quadlet files.

3. **Build image detection** — After podlet generates `.build` files, the graft step should detect which containers are built from these files and skip registry pulls for them. This avoids `podman pull` failures when the built image doesn't exist in a registry.

4. **Build-time secrets** — Support for compose `secrets:` used as build args (passed via `--secret` to podman build).

### Partial Deploy Coverage with Grafting (using systemd `[Service]` Directives)

The graft step can inject systemd `[Service]` directives into `.container` quadlet files. Quadlet passes through unknown `[Service]` directives to the generated `.service` file. This enables:

1. **`EnvironmentFile=`** — Parse compose `env_file:` blocks and write them to managed files, then inject `EnvironmentFile=` in `[Service]`. This offloads environment variable management from quadlet to systemd, avoiding issues with large or sensitive environment blocks.

2. **Healthcheck integration** — Compose `healthcheck:` could be translated into systemd health monitoring via `WatchdogSec=` and `ExecStartPost=` hooks, providing restart-on-unhealthy behavior natively.

3. **Dependency ordering** — `depends_on:` conditions (service_healthy, service_started, etc.) could be mapped to `ExecStartPre=` commands that poll dependent services before starting.

4. **Startup timeout tuning** — Map compose `deploy.restart_policy` and `stop_grace_period` to systemd `TimeoutStartSec=`, `TimeoutStopSec=`, `RestartSec=`, and `Restart=` directives.

### Other Gaps

- **`configs:` compose section** — Similar to secrets but for non-sensitive configuration files. Could use the same direct bind mount pattern.
- **Long syntax for secrets** — `target:` in service-level secret references (for custom mount paths) is partially supported via `SecretRef.Target`. Full implementation needs end-to-end testing.
- **Swarm mode compatibility** — Swarm-specific compose extensions (deploy modes, constraints, replicas) are not supported.

### Future Security Improvements

- **Tmpfs-backed secrets via `LoadCredential`** — Currently secrets are bind-mounted directly from managed files on disk. Consider generating a companion `.service` service unit file (not a `.container` quadlet) that uses systemd `LoadCredential=` in `[Service]` combined with `Volume=%d/<name>` to mount secrets from systemd's RAM-backed credential directories (`/run/credentials/`). This would keep secret values in tmpfs memory rather than on persistent storage. Initial implementation attempted this using quadlet's `[Service]` pass-through, but `LoadCredential=` + `Volume=` with credential paths didn't integrate correctly with quadlet's container lifecycle. A standalone `.service` file could bypass quadlet entirely for credential setup.

---

## 🗺️ Long-Term Roadmap Goals

For reference, these are tracked in [ROADMAP.md](./ROADMAP.md).
