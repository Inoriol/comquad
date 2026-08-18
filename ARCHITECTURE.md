# Architecture & Technical Design

This document details the internal design, component mapping, and execution lifecycle of **comquad**. Read this to understand how your Docker Compose components translate safely into native systemd Quadlets.

## 🔄 Execution Lifecycle

When you run `comquad up`, the engine moves your configuration through a two-step pipeline:

1. **Transpile** — `compose2quadlet.TranspileFile()` loads the compose file via compose-go/v2, maps all fields to structured `QuadletUnit` objects, applies opinionated transforms (prefix, references, SELinux, labels, port offset, auto-update, install section, default network, network aliases), and resolves secrets, Dockerfile FROM lines, and volume paths.
2. **Reconcile & Deploy** — compute a change plan (`reconcile.Compute`), optionally show a diff and ask for confirmation, apply changes (`reconcile.Apply`), register in state file, pull images per strategy, reload daemon, and start/restart only affected units via D-Bus.

### Dry Run Mode (`--dry-run`)

When `comquad up --dry-run` is used, `TranspileFile()` is called with `WithDryRun()` to prevent side effects (no secret files written to disk). After transpilation, comquad builds a change plan via `reconcile.Compute` (read-only, no writes) and prints:

- **Image pull actions** that *would* be taken per container, based on the pull strategy and whether images exist locally
- For each file, a **color-coded diff** — new files as full content, changed files as a unified diff, and removed files (services dropped from `compose.yaml`) as a removal diff

The deploy step is skipped entirely: no files are written to the systemd directory, no baseline is updated, no state is registered, and no units are started.

## 🔄 Reconcile & Change Detection

On every `up`, comquad reconciles the freshly generated quadlet units against what is already deployed instead of blindly overwriting. The `internal/reconcile` package powers this:

- **Baseline** — each successful deploy stores the *pure generated* content of every quadlet file under `$XDG_DATA_HOME/comquad/baseline/<project>/`. The baseline is what `compose.yaml` produced last time, never the merged result.
- **Three-way merge** — `MergeUnit(base, disk, new)` does a directive-level merge. `disk` is the on-disk file (baseline + manual `edit` changes), `new` is the fresh generation. Per directive key it resolves: unchanged / user-changed / compose-changed / both-changed (conflict → user wins + warning) / added / removed.
- **Plan / Apply split** — `Compute(...)` builds a read-only `Plan` (per-file status, old/new content, conflicts); `Apply(...)` writes files atomically, updates the baseline, removes stale files, and restores already-applied targets/baselines if a later operation fails. `Up` shows `Plan.Diff()` (a color-coded unified diff) and asks for confirmation before calling `Apply`.
- **Selective restart** — only created units are started and only changed containers/images are restarted; units for dropped services are stopped before `daemon-reload` forgets them.
- **Baseline lifecycle** — rewritten on each successful `up`, removed on `down`, cleared by `regenerate`, and restored to its previous content on a failed `up` (`Apply` and `rollbackDeploy` restore files and baselines touched by failed operations instead of deleting the whole project).

`--no-diff` skips the diff and confirmation. On a first deploy (or after `regenerate`) there is no baseline, so reconciliation falls back to a 2-way comparison (overwrite + warning) that cannot distinguish manual edits.

## 📦 Project Directory Structure

The codebase is organized cleanly into domains matching the execution lifecycle steps:

```text
cmd/comquad/           # CLI entry point: main.go + per-command files (up.go, down.go, logs.go, …)
compose2quadlet/       # Compose → Quadlet transpilation library (in-tree sub-module)
internal/deploy/       # Systemd D-Bus communication, target directories, state tracking,
                       # and the SystemdClient / StateStore interfaces used for testing
internal/logger/       # Colorized logging with quiet/verbose tiers
internal/reconcile/    # Change detection: 3-way merge (MergeUnit), plan/apply split,
                       # baseline storage, and unified-diff rendering
internal/orchestrator/ # The engine wiring all packages: orchestrator.go (core/Up), down.go,
                       #   images.go (pull/dry-run), pipeline.go (helpers), plus
                       #   per-command files (lifecycle, logs, exec, view, edit, etc.)
```

## 🧩 compose2quadlet Library

comquad's transpilation is powered by the in-tree [compose2quadlet](./compose2quadlet/) Go package. It loads compose files via compose-go/v2, maps services/networks/volumes/secrets/configs to structured `QuadletUnit` objects, applies opinionated transforms, and serializes to quadlet ini format.

For its internal architecture and package layout, see:
- [compose2quadlet/ARCHITECTURE.md](./compose2quadlet/ARCHITECTURE.md)
- [compose2quadlet/README.md](./compose2quadlet/README.md)

### Post-transpile Processing

After `TranspileFile()` returns, comquad applies one fix before writing units to disk:

- **`stripServiceName`** — Removes `ServiceName=` from container units. Without this, quadlet names the systemd unit after the compose service name (e.g. `db.service`) instead of the file-prefixed name (`cq-<project>-db.service`), breaking comquad's unit lookup.

The library handles the rest through its opinionated transforms:
- **`ApplyContainerName`** — Supplies `ContainerName=<project>-<service>` when no explicit Compose `container_name` is present (e.g. `nextcloud-redis-mariadb-db`), matching Docker Compose's default naming convention.
- **`ApplyNetworkAliases`** — Injects `NetworkAlias=<service>` and `NetworkAlias=<project>-<service>` so services can resolve each other by compose service name and project-qualified name within compose networks.
- **`.image` references are kept** — Containers reference `.image` files via `Image=<name>.image` (not resolved). The quadlet generator follows the chain through to the `.image` unit's fully-qualified `Image=` directive, which satisfies `AutoUpdate=registry`'s requirement for fully-qualified references.
- **External resources are preserved** — External networks and volumes are not generated as Quadlets; references retain their configured Podman resource names instead of being rewritten to managed `cq-<project>-` units.
- **`SECTION SPACING`** — Serialization now adds blank lines between non-empty sections for readability, matching the format produced by the previous podlet-based pipeline.

## 📊 Ps Command

The `ps` command shows container runtime status in `docker compose ps` style. It queries Podman for container data and merges it with systemd D-Bus unit state.

**Data sources:**

1. **Podman** — Runs `podman ps --filter "label=com.comquad.managed=true" --filter "label=com.comquad.project=<name>" --format json` (or `-a` flag for exited containers). Parses JSON to extract container name, image, command, state, status, ports, networks, mounts, exit code, and timestamps. Runs `podman inspect <container>` for each container to extract `Config.ExposedPorts`.
2. **D-Bus** — Queries `ListUnitsByNames()` targeting only the project's container units. Merges `ActiveState` and `SubState` into each container record.

**Output format:**

Uses `go-pretty/table` for auto-width columns with `StyleLight` borders. The COMMAND column is capped at 30 characters with automatic text wrapping for long commands.

```
┌────────────────┬────────────────────────────────────┬────────────────────────────────┬─────────┬─────────┬───────────────┬──────────────────────────────┐
│ NAME           │ IMAGE                              │ COMMAND                        │ SERVICE │ CREATED │ STATUS        │ PORTS                        │
├────────────────┼────────────────────────────────────┼────────────────────────────────┼─────────┼─────────┼───────────────┼──────────────────────────────┤
│ nexcloud-db    │ docker.io/library/mariadb:10.5     │ --transaction-isolation=READ-C │ db      │ 17m ago │ Up 17 minutes │ 3306/tcp                     │
│                │                                    │ OMMITTED --binlog-format=ROW   │         │         │               │                              │
│ nexcloud-nc    │ docker.io/library/nextcloud:apache │ apache2-foreground             │ nc      │ 17m ago │ Up 17 minutes │ 80/tcp, 0.0.0.0:2080->80/tcp │
│ nexcloud-redis │ docker.io/library/redis:alpine     │ redis-server                   │ redis   │ 17m ago │ Up 17 minutes │ 6379/tcp                     │
└────────────────┴────────────────────────────────────┴────────────────────────────────┴─────────┴─────────┴───────────────┴──────────────────────────────┘
```

Exited containers show `Exited (<code>) <time>` in the status column. Dead containers show `Dead`. Ports are formatted as `host_ip:host_port->container_port/protocol` for published ports, and `port/protocol` for exposed ports (container-only, no host binding). Exposed ports are listed first, followed by published ports. Created time uses relative format (`just now`, `5m ago`, `2d ago`, or `Jan 02 2006` for older entries).

Containers are sorted: running first (by name), then exited (by name), then other states (by name). This ensures `ps -a` groups exited containers together at the bottom of the table.

**Flags:**

* `-a, --all` — Include exited/dead containers (uses `podman ps -a`)

**Testability:** `Orchestrator.listContainers` is an injectable function field (`func(projectName string, all bool) ([]ContainerInfo, error)`), allowing tests to provide canned container data without a live Podman daemon.

## 👁️ View Command

The `view` command (also available as `overview`) provides two modes of inspection:

**Project view** (no service argument): displays a clean relational overview using `go-pretty/table` with auto-width columns. The output is split into two tables:
- **SERVICES** — each container with its status, short image name, attached networks, and volumes. Image names are resolved through `.image` quadlet files when containers reference them, showing human-readable names (e.g. `mariadb:10.5` instead of `docker.io/library/mariadb:10.5`).
- **RESOURCES** — all images, networks, and volumes with copy-pasteable names (e.g. `db.image`, `dbnet.network`). These names can be used directly as arguments to `comquad view <name>`.

The header shows Project, Source, and aggregate Status (`healthy` when all containers are running, `stopped` when none are, `degraded` when only some are).

**Unit file view** (with service argument): resolves the quadlet file using five matching patterns (`web` → `cq-myapp-web.container`, `cq-myapp-web`, `cq-myapp-web.service`, `cq-myapp-web.container`, or `myapp-web`). Non-container resources support short-name matching (e.g. `db.image`, `dbnet`, `dbnet.network`, `cq-myapp-db-image.service`). Reads the file and prints its contents prefixed with a `── <filename> ──` header.

## ✏️ Edit Command

The `edit` command provides two modes of file editing:

**Project edit** (no service argument): resolves all `.container`, `.network`, `.volume`, `.image`, and `.build` quadlet files for a project and opens them in `$EDITOR` (falls back to `findDefaultEditor()` which probes `editor`, `nano`, `vim`, then `vi`). `$EDITOR` is split on whitespace, so values like `"vim -o"` or `"code --wait"` work correctly. After the editor exits, comquad compares file contents and auto-reloads systemd, restarting any changed container units.

**Unit file edit** (with service argument): resolves a single quadlet file using the same matching patterns as `view`, opens it in the editor, and reloads systemd if changes were detected.

The `--no-reload` flag opens files without triggering a systemd daemon reload or unit restart.

Unit resolution shares the same matching logic as `view`, using `MatchFirstContainer` and `MatchQuadletResource` helpers that iterate over `state.Files` from `projects.json`.

## 📋 Logs Command

The `logs` command queries systemd D-Bus to determine each unit's state and filters output accordingly:

**Running units** (`ActiveState == "active"`): retrieves the `InvocationID` property via D-Bus `GetUnitProperties` and passes it to `journalctl --invocation=<hex>`, showing only logs from the current invocation.

**Stopped / failed units**: no filter is applied, showing full historical logs.

Service name matching uses the same multi-pattern logic as `view` and `edit`: exact file name, name without extension, name with `.service` suffix, short name (after stripping `cq-<project>-` prefix), internal Podman name (after stripping `cq-` prefix), or `ContainerName=` directive from the unit file. `MatchAllContainers` returns all matching files per argument, allowing a single arg like `web` to match multiple services.

### Flags

- `--tail <N>` — Limit output to the last N lines
- `--since <time>` — Show logs since a specific time (e.g. `10m`, `2024-01-01 12:00:00`)
- `-t, --time` — Display timestamps in RFC3339Nano format (docker compose compatible)

Logs from multiple units are collected via `journalctl --output=json`, parsed, sorted by `__REALTIME_TIMESTAMP`, and rendered in chronological order. Each line is prefixed with `[<unit-name>]` to identify its source.

## 🔄 Lifecycle Commands (Start, Stop, Restart)

The `start`, `stop`, and `restart` commands manage the runtime state of deployed projects without touching quadlet files or triggering daemon-reload. They operate directly via D-Bus.

**Service resolution:** All three commands accept optional `[service ...]` positional arguments. When provided, they use `MatchAllContainers` to resolve service names to unit names. When omitted, all `.container`, `.image`, and `.build` files from the project's state are started/stopped/restarted.

**Start** — Iterates over resolved unit names and calls `StartUnit` via D-Bus. Reports per-unit status messages.

**Stop** — Iterates over resolved unit names and calls `StopUnit` via D-Bus, then verifies all units are no longer active (same verification as `down`).

**Restart** — Iterates over resolved unit names and calls `RestartUnit` via D-Bus, which tears down and recreates the unit cleanly.

All three commands require the project to exist in `projects.json` state. They share a `resolveUnits()` helper that looks up the project state, matches service names, and deduplicates results.

**Flags:** All three commands accept `--dry-run` to preview which units would be affected without making changes.

## 🗑️ Down Command

The `down` command performs a complete teardown of a deployed project in seven steps:

1. **Stop units** — Stops all container, image, and build units via systemd D-Bus `StopUnit`, then verifies all container units are no longer active (aborting if any remain). Network and volume units are also stopped.
2. **Remove quadlet files** — Deletes all `.container`, `.network`, `.volume`, `.image`, and `.build` files from the systemd target directory.
3. **Reload daemon** — Triggers `daemon-reload` via D-Bus so systemd forgets the removed units and releases its references to networks and volumes.
4. **Remove networks** — Lists all Podman networks with both `com.comquad.managed=true` and the matching `com.comquad.project` label, then removes them via `podman network rm`.
5. **Remove volumes (opt-in)** — When the `-d, --delete-volumes` flag is provided, lists all Podman volumes with both `com.comquad.managed=true` and the matching `com.comquad.project` label, then removes them via `podman volume rm`. Volumes are opt-in because they may contain persistent data.
6. **Unregister project** — Removes the project entry from `projects.json` state file.
7. **Remove auxiliary state** — Deletes the project's baseline (`$XDG_DATA_HOME/comquad/baseline/<project>`), secrets, and build-cache directories.

**Usage:**

```bash
comquad down          # stops containers, removes networks, removes quadlet files
comquad down -d       # also removes Podman volumes
comquad down -y       # skip confirmation prompt
comquad down --dry-run # preview what would be removed without making changes
```

The `down` command prompts for confirmation when stdin is a terminal. This prevents accidental teardown. Confirmation is skipped when `--dry-run`, `--yes`/`-y`, or when stdin is not a terminal (piped/non-interactive).

**Flags:**
* `-d, --delete-volumes` — Also remove Podman volumes
* `-y, --yes` — Skip confirmation prompt
* `--dry-run` — Show what would be removed without actually removing anything

## 🐳 Exec Command

The `exec` command runs a command inside a running container via `podman exec`. It requires a single service argument and allocates a TTY by default (like `docker compose exec`).

**Service resolution:** Uses `MatchAllContainers` to resolve the service name to a container quadlet file. `exec` uses the unit's explicit `ContainerName=` when present, otherwise derives the name from the base filename by stripping the `cq-` prefix and `.container` suffix (e.g. `cq-myapp-web.container` → `myapp-web`). Interactive execution keeps stdin open and allocates a TTY by default. If the service matches multiple containers, an error is returned listing the ambiguous matches.

**Flags:** `-u/--user` sets the user inside the container, `-t/--tty` controls TTY allocation (default `true`). The command is passed directly to `podman exec`, which handles `--` flag separation.

## 🔄 Regenerate Command

The `regenerate` command restores the state file by scanning Podman for managed resources. It is the foundation of comquad's self-healing state management.

**Discovery pipeline:**

1. Queries Podman for all containers with label `com.comquad.managed=true`
2. Queries Podman for all networks with label `com.comquad.managed=true`
3. Queries Podman for all volumes with label `com.comquad.managed=true`
4. Groups all resources by their `com.comquad.project` label value
5. Resolves quadlet files in the systemd target directory matching `cq-<project>-*.container`, `*.network`, `*.volume`, `*.image`, `*.build`
6. Writes the reconstructed state to `projects.json`
7. Clears the baseline directory for each discovered project — the reconstructed state has no baseline, so the next `up` falls back to a 2-way reconcile (compose wins + warning)

**Flags:**

* `--force` — Required to overwrite existing state (safety guard)
* `--dry-run` — Preview what would be regenerated without writing the state file. Can be combined with `--force` to safely inspect what `regenerate` would do.

**Usage:**

```bash
comquad regenerate --force              # regenerate state from Podman labels
comquad regenerate --force --dry-run    # preview without writing
```

## 💾 State & File System Management

### State File Location

`comquad` keeps track of active, managed projects in a light JSON database.

* **Default:** `~/.local/share/comquad/projects.json`
* **Overridden by:** `$XDG_DATA_HOME/comquad/projects.json`

### State File Format

Each project entry contains:

```json
{
  "project_name": "myproject",
  "source_path": "/home/user/projects/myproject",
  "files": [
    "/path/to/cq-myproject-web.container",
    "/path/to/cq-myproject-web.image",
    "/path/to/cq-myproject-default.network"
  ],
  "resources": {
    "containers": ["myproject-web"],
    "networks": ["myproject-default-network"],
    "volumes": ["myproject-data"],
    "images": ["myproject-web"],
    "builds": []
  }
}
```

* **project_name** — The Comquad project name
* **source_path** — Path to the compose file directory (empty when restored via `regenerate`)
* **files** — List of quadlet file paths in the systemd target directory. Includes `.container`, `.image`, `.network`, `.volume`, and `.build` files.
* **resources** — Podman resources tracked for the project (`containers`, `networks`, `volumes`, `images`, `builds`). Populated by both `up` (derived from quadlet filenames) and `regenerate` (discovered from live Podman labels). When `up` re-registers, it overwrites resource info with the current file set.

The state file is written atomically: comquad writes to a `.tmp` file in the same directory and renames it, so a crash mid-write never produces a corrupt state file.

### Target Systemd Directories

Quadlet configurations are copied into paths dictated by your execution context:

* **Rootless Mode (Default user):** `~/.config/containers/systemd`
* **Root Mode (UID 0 / sudo):** `/etc/containers/systemd`

---

## 📋 Compose Format Implementation Details

`comquad` accepts standard Docker Compose v3 files supporting `services`, `networks`, and `volumes`.

All compose-to-quadlet mapping, Dockerfile normalization, secret resolution, and opinionated transforms (prefix, references, SELinux, labels, port offset, auto-update, install section, default network, network aliases) are handled by the [compose2quadlet](./compose2quadlet/) library (in-tree sub-module).

### Automatic Behaviors & Opinionated Transforms

To ensure the transition to Quadlets is frictionless, the internal engine enforces several rules:

* Relative volume host paths are automatically fully-qualified to absolute paths.
* When SELinux is detected (via `/sys/fs/selinux/enforce`), `Volume=` directives in generated `.container` files get `,z` appended and `Mount=` directives get `relabel=shared` appended (`Mount=type=bind,...` → `Mount=type=bind,...,relabel=shared`). Idempotent — skips if `:z`, `:Z`, or `relabel=` already present.
* A default bridge network (`cq-default`) is implicitly injected whenever a container lacks an explicit `Network=` directive. This includes cases where user-defined networks exist but a service has no `networks:` key.
* Generated containers default to the `<project>-<service>` naming blueprint; an explicit Compose `container_name` is preserved.
* `NetworkAlias=` is injected into every `.container` file so services can resolve each other by service name and `ContainerName=` value within compose networks.
* An identifying label (`com.comquad.project`) is attached to all generated units that support labels (`.container`, `.network`, `.volume`, `.build`).
* A `com.comquad.managed` label is attached to those same units to indicate comquad management. `.image` units have no `[Container]`-style `Label=` directive, so they are identified by filename rather than label.
* Unprefixed public images default seamlessly to standard Docker Hub (`docker.io/library/`).
* In rootless mode, privileged ports (≤ 1024) are automatically offset by `ROOTLESS_PORT_OFFSET` (default 2000). Internal port conflicts within a project are resolved by incrementing.

### Build Blocks

`build:` blocks are handled natively by compose2quadlet which generates `.build` quadlet files, patches Dockerfile `FROM` lines for consistent image resolution, and updates the corresponding `.container` file's `Image=` to reference the `.build` file.

### Quadlet Feature Injections

You can inject native Quadlet behaviors via labels. For example, to prevent auto-updates on a specific container image, set `comquad-no-autoupdate`:

```yaml
services:
  web:
    image: nginx
    labels:
      comquad-no-autoupdate: "true"
    # labels:                        # Also supported (list format):
    #   - comquad-no-autoupdate=true
```

### Image Quadlet Generation

comquad automatically generates `.image` quadlet files for every service container via compose2quadlet. The library maps compose `pull_policy`, `platform`, and `image` fields to the appropriate `[Image]` section directives.

| Compose field | `.image` directive | Notes |
|---|---|---|
| `image: nginx:latest` | `Image=` | Uses the compose-go/v2 normalized image |
| `pull_policy: always` | `Policy=always` | `if_not_present` aliased to `missing` |
| `pull_policy: missing` | `Policy=missing` | Default behavior |
| `platform: linux/amd64` | `OS=linux`, `Arch=amd64` | |
| `platform: linux/arm64/v8` | `OS=linux`, `Arch=arm64`, `Variant=v8` | |

Unsupported `pull_policy` values (`build`, `daily`, `weekly`, `every_*`) are logged as a warning and omitted. Hardcoded defaults `Retry=3` and `RetryDelay=5s` are always applied.

The `.container` file's `Image=` directive is updated to reference the `.image` file (e.g. `Image=cq-myapp-web.image`), which causes the quadlet generator to create a systemd dependency between the container unit and the image unit.

`.image` files are written to the systemd target directory, registered in `projects.json`, and picked up by `collectProjectFiles()` in the deploy pipeline. During deployment, `startUnits()` starts `.image` units before `.container` units. If the system's quadlet generator does not support `.image` files, startup failures are logged as warnings and deployment proceeds — the manual pull via `handleImages()` provides a fallback.

### Secrets Management

comquad delegates compose `secrets:` processing to compose2quadlet, which translates them into quadlet-native directives. External secrets produce `Secret=` for Podman's native secret store. File-based and environment-based secrets produce `Volume=` mounts in `[Container]` at `/run/secrets/<name>`.

### Partial Deploy Coverage

compose2quadlet can inject not only quadlet-specific directives (`Secret=`, `Volume=`) but also arbitrary systemd `[Service]` directives. This enables future features including `EnvironmentFile=`, `ExecStartPre=/ExecStartPost=`, `TimeoutStartSec=`, and custom restart policies.

## 🧪 Testing Architecture

The orchestrator package was historically untestable because it constructed `SystemdManager` and `StateManager` directly inside every method. Two interfaces in `internal/deploy/interfaces.go` solve this:

- **`SystemdClient`** — all nine D-Bus methods used by the orchestrator (`StartUnit`, `StopUnit`, `RestartUnit`, `ReloadDaemon`, `WaitForUnit`, `ListUnitsByNames`, `ListAllUnits`, `GetInvocationID`, `Close`). The concrete `SystemdManager` satisfies this interface.
- **`StateStore`** — all state operations used by the orchestrator (`GetProject`, `GetStateFilePath`, `ListProjects`, `RegisterProject`, `UnregisterProject`, `Save`). The concrete `StateManager` satisfies this interface. `GetProject` replaces direct `Projects[name]` map access, making the interface satisfiable without exposing the map.

`Orchestrator` holds four factory fields instead of calling the constructors directly:

```go
newState       func() (deploy.StateStore, error)
newSystemd     func() (deploy.SystemdClient, error)
listContainers func(projectName string, all bool) ([]ContainerInfo, error)
newJournalCmd  func(name string, args ...string) *exec.Cmd
```

`NewOrchestrator` wires in the real implementations. Tests override these fields with in-memory fakes (`mockStateStore`, `mockSystemdClient`) that record calls and return canned responses, enabling full unit-test coverage without a live D-Bus or Podman daemon.

### Project Deployment Helper

The `ensureProjectDeployed()` helper encapsulates the common pattern of creating a state manager, fetching the project, and failing if it does not exist. It returns `(StateStore, ProjectState, error)` to support both state access and operations like `UnregisterProject`. Eight methods use this helper: `Ps`, `Down`, `View`, `Edit`, `Exec`, `Logs`, `FollowLogs`, and `resolveUnits`.

### Image Normalization

Image normalization (e.g., `nginx` → `docker.io/library/nginx`) is handled natively by compose-go/v2, the canonical compose specification implementation used by compose2quadlet.

### CI Pipeline & Build

Automated testing runs via `.github/workflows/test.yml` on every push and PR to `main`:

- **Build & vet** — `go build ./...` and `go vet ./...`
- **Short tests** — `go test -short` (skips tests requiring external binaries)
- **Race detector** — `go test -race` on all packages
- **Coverage** — `go test -cover` with per-package and total coverage report

Makefile targets (`make test-unit`, `make test-race`, `make test-cover`, `make test-short`) provide
the same commands locally.

The `captureStdout` helper in `internal/orchestrator/dryrun_test.go` uses a `sync.Mutex` to serialize
`os.Stdout` redirection, making it safe to use alongside `t.Parallel()` — only the capture window
serializes, not the entire test.

## 📋 Follow Logs on Deploy

When `comquad up -f` is used, after successfully deploying all units the CLI captures the current timestamp and streams all journal logs for every project unit (containers, networks, volumes, images, and builds) from that point onward. This emulates the default `docker compose up` behavior (without `-d`), keeping the terminal attached to live output until interrupted with Ctrl+C.

The deployment timestamp is captured after image handling completes but before `daemon-reload` and unit starts, ensuring no startup logs are missed.

The follow mode uses JSON output with a buffered flush (500ms interval) to maintain chronological order and supports the `-n/--tail` and `-t/--time` flags via the `FollowLogs()` function.

## 📊 Output Levels

comquad has three output modes, controlled by flags on the root command:

| Mode | Flag | What's shown |
|---|---|---|
| Normal (default) | *(none)* | Operational messages: pipeline stage progress, unit starts/stops, deploy success, errors |
| Verbose | `-v` / `--verbose` | All of the above plus every pipeline transformation |
| Quiet | `-q` / `--quiet` | Errors only (stderr). All other output is suppressed. Useful in scripts. |

The `Up` pipeline prints progress indicators at each stage via `logger.Action()`:
- "Reading compose file..."
- "Transpiling compose configuration..."
- "Generating quadlet files..."
- "Handling images..."
- "Starting services..."

These appear in normal mode (blue), distinct from verbose-only `logger.Info()` output.

`--quiet` takes precedence over `--verbose`. `logger.Error(...)` always writes to stderr regardless of either flag.

The logger lives in `internal/logger/` and exposes four tiers:
- **`logger.Print(msg)`** — normal operational output, suppressed by `--quiet`
- **`logger.Action/Success/Warn(msg)`** — user-facing actions and confirmations (blue/green/yellow), shown by default, suppressed by `--quiet`
- **`logger.Info(msg)`** — verbose-only pipeline internals, requires `-v`
- **`logger.Error(msg)`** — always to stderr, never suppressed

Colors use ANSI codes (green=success, cyan=info, yellow=warning, red=error, blue=action) and are disabled when `NO_COLOR` is set.

When `-v` / `--verbose` is enabled, comquad additionally logs every transformation applied during the deployment pipeline:

**Deploy stage** logs:
- `Removing network: <name>` — network removal
- `Removing volume: <name>` — volume removal

Network and volume removal errors are always printed to stderr regardless of verbose mode, so cleanup failures are never silently dropped.

**Error output:** `logger.Error(...)` always writes to stderr regardless of the verbose setting. Only internal pipeline details (`logger.Info`) are gated behind `-v`.

## 🧪 Integration Testing

Integration tests verify the full end-to-end lifecycle of comquad against a real
systemd instance and real Podman daemon. They complement the
unit tests (which mock D-Bus and state via interfaces) by exercising the complete
pipeline from `compose` → quadlet files → running containers.

### Test Environment

Tests run inside a privileged Podman container with systemd as PID 1. This gives
each test run a fully isolated, reproducible environment with a real D-Bus session,
real cgroup hierarchy, and real systemd unit activation — without touching the host.

The test image is defined in `tests/integration/Containerfile` and baked ahead of
time (never installed at test runtime) with all required dependencies:

- `golang` — to compile and run integration test binaries
- `podman` — container runtime
- `systemd` — PID 1, D-Bus, unit management
- `sudo`, `shadow-utils`, `slirp4netns`, `fuse-overlayfs` — rootless support

The `comquad` binary is pre-built on the host (`make build`) and mounted read-only
into the container via the workspace volume, so the container never rebuilds it.

A non-root user (`testuser`) is pre-created with `/etc/subuid` and `/etc/subgid`
entries. Linger is enabled by writing `/var/lib/systemd/linger/testuser` directly
(no runtime `loginctl` call needed), so the systemd user instance starts correctly
for rootless test scenarios.

### Test Structure

```text
tests/
  integration/
    helpers/
      binary.go        # Invoke comquad binary, capture stdout/stderr/exit code
      compose.go       # Write temp compose files, reusable compose templates
      podman.go        # Inspect Podman containers, networks, volumes
      selinux.go       # SELinux detection helpers for conditional test skipping
      state.go         # Read and assert projects.json state file contents
      systemd.go       # Poll and assert systemd unit states via systemctl
    testdata/          # Static compose files and Dockerfiles for complex scenarios
    up_down_test.go    # Core up/down lifecycle, idempotency, volume retention
    dry_run_test.go    # Dry-run isolation: no files written, no state registered
    lifecycle_test.go  # start/stop/restart command flows
    logs_test.go       # Log retrieval for running and stopped units
    exec_test.go       # podman exec command tests
    exec_ambiguous_test.go # Ambiguous service matching validation
    rootless_test.go   # Rootless mode: port offsetting, target directory, user instance
    selinux_test.go    # SELinux :z label injection (quadlet files and runtime mounts)
    view_edit_test.go  # view/edit command tests
```

### Test Design Decisions

All tests use `--name <project>` with `comquad up` to explicitly specify the project name,
decoupling test behavior from the directory name. The `WriteCompose` helper returns the
project name parsed from the compose `name:` field, ensuring tests always use the correct
name for both `up` and `down` calls. This prevents fragile dependencies on `t.TempDir()`
naming conventions.

SELinux tests use `helpers.SELinuxPresent(t)` for skip conditions to detect SELinux
mount availability, not enforcement mode, since comquad's `:z` injection triggers on
presence detection via `/sys/fs/selinux/enforce` file content.
