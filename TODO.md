## 🗺️ Roadmap & Next Steps

For long-term goals, refer to the [roadmap](./ROADMAP.md).

### Follow-ups

- **Value-level merge** — Multi-value directives (`Environment=`, `Volume=`, `PublishPort=`) are merged at key granularity; independent additions to the same key currently conflict instead of being merged per line.
- **Diff rendering** — The unified diff implementation is homegrown; it could be replaced with `go-udiff` if richer output is ever needed.

### Possible Security Improvements

- **Tmpfs-backed secrets via `LoadCredential`** — Currently, secrets are bind-mounted directly from managed files on disk. Consider generating a companion `.service` unit (rather than a `.container` Quadlet) that uses systemd `LoadCredential=` in `[Service]`, combined with `Volume=%d/<name>`, to mount secrets from systemd's RAM-backed credential directories (`/run/credentials/`). This would keep secret values in tmpfs memory rather than on persistent storage. An initial implementation using Quadlet's `[Service]` pass-through did not integrate `LoadCredential=` and `Volume=` with credential paths correctly in the container lifecycle. A standalone `.service` file could bypass Quadlet entirely for credential setup.

## Respecting registries

- **Respecting registries.conf** — Currently we assume that if registry is not specified, it will be pulled from docker hub. I think it's right behavior (fool-proofing), but giving option to respect registries.conf file may be also good. Maybe enviromental variable? Theoretically no change or read of registries needed from comquad side, just skipping registry normalizing with this option.

### Quadlet Directives in Compose Files

- **X-extension** — ✅ Implemented. Supports `x-container`, `x-image`, `x-build`, `x-network`, `x-volume`, and `x-systemd` (for `[Service]` and `[Unit]` sections). X-extensions override compose-generated directives. Validates directive names and warns on unknown directives.

- **Timers** — Service-level timer support via `x-timer`. Design decisions:
  - **Target**: `Target: "build"|"service"` (defaults to `"build"` if service has `build:`, else `"service"`)
  - **Restart behavior**: `RestartService: true` adds `BindsTo=` to container unit, causing automatic restart when build completes
  - **Warnings**:
    - Build timer without `RestartService: true` warns that container won't use rebuilt image
    - `Target: "service"` with `build:` present warns about stale image
  - **Structure**:
    ```yaml
    services:
      web:
        build: .
        x-timer:
          OnCalendar: "daily"
          Persistent: true
          Target: "build"
          RestartService: true
    ```
  - **Generated units**: `<service>.timer` with `[Timer]` section, targeting `.build` or `.service` unit
  - **Restart mechanism**: Uses `BindsTo=` (native systemd dependency) rather than `ExecStartPost=`
  - **Multiple timers**: Not supported (keep it simple)

- **Questionable future feature: Round-trip** — Make it possible to convert a project back into a Compose file.

### Paths

- **Consider shortening paths** — Consider using Systemd Specifiers for path if appliable
- **More elaborate handling in labels on SElinux volumes** - Consider implementing :Z label for shared volumes and keeping :z for non-shared

