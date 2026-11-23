# compose2quadlet

Go library that transpiles Docker Compose files into Podman Quadlet unit files.

**Not a CLI tool.** This is a library designed to be consumed by [comquad](https://github.com/Inoriol/comquad) and other tools that need programmatic compose-to-quadlet conversion.

Heavily inspired by [podlet](https://github.com/containers/podlet), but designed for the Go ecosystem.

## Motivation

Replaces the former `podlet` binary dependency with a native Go library. Eliminates the fragile text-based pipeline (preprocess → strip → transpile → cook → graft) with structured, type-safe quadlet output.

## Usage

**Quick path** — `TranspileFile` loads the compose file with `.env` resolution and transpiles in one call:

```go
import (
    c2q "github.com/Inoriol/comquad/compose2quadlet"
    "github.com/Inoriol/comquad/compose2quadlet/serialization"
)

func main() {
    units, _ := c2q.TranspileFile("compose.yaml",
        c2q.WithProjectName("myapp"),
        c2q.WithPortOffset(10000),
        c2q.WithAutoUpdate(),
        c2q.WithLabels(map[string]string{
            "com.myorg.managed": "true",
            "com.myorg.project": "myapp",
        }),
    )

    // Write all units to a directory
    serialization.WriteUnits("/etc/containers/systemd", units)
}
```

**Full control** — load the compose-go project yourself and call `Transpile`:

```go
import (
    c2q "github.com/Inoriol/comquad/compose2quadlet"
    "github.com/Inoriol/comquad/compose2quadlet/serialization"
    "github.com/compose-spec/compose-go/v2/cli"
)

func main() {
    opts, _ := cli.NewProjectOptions(
        []string{"compose.yaml"},
        cli.WithOsEnv,
        cli.WithDotEnv,
    )
    project, _ := opts.LoadProject(context.Background())

    units, _ := c2q.Transpile(project,
        c2q.WithProjectName("myapp"),
        c2q.WithPortOffset(10000),
        c2q.WithAutoUpdate(),
        c2q.WithLabels(map[string]string{
            "com.myorg.managed": "true",
            "com.myorg.project": "myapp",
        }),
    )

    for _, u := range units {
        fmt.Printf("%s.%s\n", u.Name, u.Type)
    }

    // Serialize to ini format for writing to disk
    text := serialization.Marshal(units[0])
    fmt.Println(text)
}
```

## Packages

- **Root** — `Transpile()`, `TranspileFile()`, `ParseVersion()`, core types (`QuadletUnit`, `Section`, `Directive`), option constructors
- **`mapper/`** — Field mapping: compose-go `ServiceConfig` / `*Project` → quadlet directives and units
  - `container.go`, `healthcheck.go`, `security.go`, `ports.go` — per-service container directives
  - `service.go` — systemd `[Service]` resource-control and restart directives
  - `unit.go` — `depends_on` → `[Unit]` dependencies and health polling hooks
  - `image.go`, `build.go` — companion `.image` and `.build` structural units
  - `network.go`, `volume.go` — top-level `.network` and `.volume` structural units
  - `jobs.go` — job container directives (Compose Spec v2.17.0+)
  - `timer.go` — job schedule → `.timer` unit conversion
  - `cron.go` — cron expression → systemd calendar format conversion
  - `secrets.go` — pre-mapping interceptor for secrets and configs; generates `.cq-secret` units for Podman secret store management
  - `dockerfile.go` — `PatchDockerfileFROM()` normalizes bare image names in Dockerfile FROM lines
  - `extensions.go` — x-extension extraction for quadlet-only directives
  - `validation.go` — directive validation for x-extensions
- **`opinionated/`** — Composable post-processing transforms (prefix, references including external resources, container name, SELinux relabeling with `:Z`/`:z` differentiation, systemd specifier shortening, labels, default network, network aliases, port offset, auto-update, install section)
- **`pseudo/`** — Pseudo-quadlet generator: converts `.cq-secret`, `.cq-image`, `.cq-build` units to `.service` units for features not natively supported by older Podman versions
- **`serialization/`** — `Marshal()`, `Write()`, `WriteUnits()`, `Unmarshal()` for ini-format serialization

Warnings from degraded or skipped mappings are sent through the `WithInfo` callback when configured. Fatal mappings still return an error. Environment-secret and patched-Dockerfile write failures are reported as degraded warnings.

## X-Extensions

X-extensions allow you to set Quadlet directives that have no Compose equivalent. They override compose-generated directives when there's a conflict.

### Supported Extensions

| Extension | Section | Example |
|-----------|---------|---------|
| `x-container` | `[Container]` | `Timezone`, `Notify`, `HealthOnFailure` |
| `x-image` | `[Image]` | `AllTags`, `AuthFile`, `CertDir` |
| `x-build` | `[Build]` | `DNS`, `Environment`, `Volume` |
| `x-network` | `[Network]` | `ContainersConfModule` |
| `x-volume` | `[Volume]` | `User`, `Group`, `Image` |
| `x-systemd` | `[Service]` and `[Unit]` | See example below |

### Example

```yaml
services:
  web:
    image: nginx
    ports:
      - "8080:80"
    
    # Container-specific quadlet directives
    x-container:
      Timezone: "Europe/Berlin"
      Notify: "healthy"
      HealthOnFailure: "restart"
    
    # Image-specific quadlet directives
    x-image:
      AllTags: true
      AuthFile: "/path/to/auth.json"
    
    # Systemd [Service] and [Unit] pass-through
    x-systemd:
      Service:
        MemoryMax: "512M"
        Restart: "on-failure"
      Unit:
        After: "network-online.target"
        Documentation: "https://example.com"

networks:
  frontend:
    driver: bridge
    x-network:
      ContainersConfModule: "/etc/containers/my.conf"

volumes:
  data:
    driver: local
    x-volume:
      User: "1000"
      Group: "1000"
```

### Validation

Unknown directives emit a warning but are still passed through. This allows forward compatibility with newer Podman versions that may add new directives.

## Pseudo-Quadlet Fallback

The library supports Podman versions back to 4.4.0 through pseudo-quadlet fallback. When targeting Podman versions below the native quadlet type requirements, the library generates pseudo-quadlet units (`.cq-image`, `.cq-build`) that are converted to standard `.service` units by the pseudo-quadlet generator.

**Native quadlet type requirements:**
- `.container`, `.network`, `.volume`: Podman 4.4.0+
- `.image`: Podman 4.8.0+ (pseudo-quadlet `.cq-image` fallback for 4.4.0-4.7.x)
- `.build`: Podman 5.2.0+ (pseudo-quadlet `.cq-build` fallback for 4.4.0-5.1.x)

**Container references:**

When pseudo-quadlets are used, container units reference the actual image instead of a quadlet reference:
- Native: `Image=web.image` (quadlet translates to dependency)
- Pseudo: `Image=docker.io/library/nginx:latest` + `After=web-image.service` + `Requires=web-image.service`

This allows containers to work with both native and pseudo-quadlet image sources without conditional logic.

**Service naming:**

Pseudo-quadlet services follow the same naming convention as native quadlet services:
- `.cq-image` → `-image.service` (matches native `.image` → `-image.service`)
- `.cq-build` → `-build.service` (matches native `.build` → `-build.service`)
- `.cq-secret` → `-secret.service` (comquad-specific)

**Usage:**
```go
units, _ := c2q.Transpile(project,
    c2q.WithPodmanVersion(c2q.Version{Major: 4, Minor: 6}),
)
units = pseudo.ConvertPseudoUnits(units)  // Converts .cq-* → .service (preserving type for state tracking)
```

**Disable fallback:**
```go
units, _ := c2q.Transpile(project,
    c2q.WithPodmanVersion(c2q.Version{Major: 4, Minor: 6}),
    c2q.WithoutPseudoQuadlet(),  // No fallback; .build causes fatal error
)
```

See [ARCHITECTURE.md](./ARCHITECTURE.md#pseudo-quadlet-generator) for details on the conversion process, and [doc/comquad-secret.md](./doc/comquad-secret.md) for the `.cq-secret` unit specification.

## Secret Handling

Secrets are handled via a pre-mapping interceptor (`PremapSecrets()`) that runs before container field mapping. The default behavior uses Podman's native secret store via `.cq-secret` pseudo-quadlet units:

**Default behavior (Podman secret store, requires Podman >= 4.5.0):**
- Environment secrets: Generates `.cq-secret` unit; during deployment, the env var value is written to the runtime secrets directory (configured via `WithRuntimeSecretsDirectory()`) and loaded via `podman secret create`
- File secrets: Generates `.cq-secret` unit that loads the file directly via `podman secret create <name> <path>`
- Container mounts: `Secret=<name>[,target=<path>,uid=<uid>,gid=<gid>,mode=<mode>]` directive mounts the secret at `/run/secrets/<name>` in the container with optional custom permissions
- Environment variable exposure: `Secret=<name>,type=env,target=<VAR>` exposes the secret as an environment variable (via `x-type: env` extension)
- Secrets are deduplicated across services — a shared secret produces only one `.cq-secret` unit
- **Secret drivers**: Supports `driver` and `driver_opts` fields for Podman secret drivers (e.g., `pass` for GPG-encrypted secrets, `shell` for script-managed secrets)

**Fallback (Podman < 4.5.0):**
- Falls back to direct `Volume=<file>:<target>:ro` mount for all secrets
- Environment secrets require `WithSecretsDirectory()` to be configured

**External secrets:**
- Use native Podman `Secret=<name>` directive (requires Podman >= 4.5)
- Support the same options (target, uid, gid, mode) and x-extensions (x-type, x-target) as managed secrets

### Secret Extensions

Secrets support x-extensions on the service reference for advanced configuration:

- `x-type: env` — Expose the secret as an environment variable instead of a file mount
- `x-target: <name>` — For `x-type: env`, sets the environment variable name; for file mounts, sets the target path

Example:
```yaml
services:
  web:
    secrets:
      - source: db_password
        x-type: env
        x-target: DB_PASSWORD
```

Generates: `Secret=cq-myapp-db_password,type=env,target=DB_PASSWORD`

### Secret Drivers

Secrets support Podman's secret driver mechanism via the `driver` and `driver_opts` fields:

```yaml
secrets:
  encrypted_secret:
    file: ./secret.txt.gpg
    driver: pass
  shell_secret:
    driver: shell
    driver_opts:
      list: /usr/local/bin/list.sh
      lookup: /usr/local/bin/lookup.sh
      store: /usr/local/bin/store.sh
      delete: /usr/local/bin/delete.sh
```

The `driver` field maps to `podman secret create --driver`, and `driver_opts` maps to `--driver-opts`. Supported drivers include `file` (default), `pass` (GPG-encrypted), and `shell` (script-managed).

## Jobs (Compose Spec v2.17.0+)

The library supports the `jobs` top-level element from the Compose specification. Jobs are one-shot containers that run to completion, unlike services which are long-running.

**Job processing:**
- `mapper.JobContainer()` — Maps job configuration to `[Container]` directives (same as services)
- `mapper.JobImages()` — Generates `.image` companion units for jobs
- `mapper.JobBuilds()` — Generates `.build` companion units for jobs
- `mapper.JobUnit()` — Generates `[Unit]` dependencies for jobs
- `mapper.Timers()` — Generates `.timer` units for scheduled jobs

**Timer generation:**
- Cron expressions are converted to systemd calendar format via `CronToSystemdCalendar()`
- Timezone support via `OnCalendar=<timezone> <calendar>` syntax
- `missed_fires: one` maps to `Persistent=true` for catch-up behavior

**Job vs Service differences:**
- Jobs skip network alias injection (no DNS discovery needed)
- Jobs skip default network attachment
- Jobs generate `.timer` units when `triggers.schedule` is present

Example:
```yaml
jobs:
  db-migration:
    image: myapp:latest
    command: python manage.py migrate
    triggers:
      manual: true

  cleanup:
    image: busybox
    command: sh -c 'find /data -mtime +30 -delete'
    triggers:
      schedule:
        - cron: "0 3 * * *"
          timezone: "Europe/Paris"
          missed_fires: one
```

## AutoNS (User Namespace Isolation)

The library supports AutoNS mode for rootful deployments that provides user namespace isolation between stacks. When enabled, one container per stack becomes the "anchor" that owns the namespace, and other containers join it.

**Enable AutoNS:**
```go
units, _ := c2q.Transpile(project,
    c2q.WithAutoNS(65535, "db"),  // base UID, anchor service name
)
```

**What it does:**
- Anchor container gets `PodmanArgs=--userns=auto:uidmapping=0:<base>:65535`
- Non-anchor containers get `PodmanArgs=--userns=container:<anchor-name>`
- Non-anchor containers get `After=` and `Requires=` for the anchor service
- All volumes get the `idmap` option

**Anchor selection** is handled by `mapper.SelectAnchor(services)` which picks the root of the `depends_on` DAG or alphabetically first service.

## Dependencies

- `github.com/compose-spec/compose-go/v2` — canonical compose parsing
- Standard library only otherwise

## Determinism

Directive ordering is deterministic: all map iteration (labels, annotations, environment, sysctls, service networks, `depends_on`, logging options, driver opts, etc.) is sorted, so transpiling the same compose file always produces byte-identical output. This makes the library safe for golden-file testing and reproducible deploys.

## License

MIT
