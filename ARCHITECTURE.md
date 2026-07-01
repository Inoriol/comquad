# Architecture & Technical Design

This document details the internal design, component mapping, and execution lifecycle of **comquad**. Read this to understand how your Docker Compose components translate safely into native systemd Quadlets.

## 🔄 Execution Lifecycle

When you run `comquad up`, the engine moves your configuration through a five-step pipeline:

1. **Preprocess** — Normalizes your `compose` yaml (resolves relative to absolute paths, sets default networks, injects project labels).
2. **Transpile** — Executes the `podlet` binary under the hood to convert the compose YAML configuration into `.container`, `.network`, and `.volume` quadlet files.
3. **Cook** — Post-processes the raw quadlet outputs. This stage prefixes files with `cq-<project>`, rewrites cross-unit references so services can communicate, injects `NetworkAlias=` for DNS resolution, injects `com.comquad.managed` and `com.comquad.project` labels on all files, and applies rootless port offsets where needed.
4. **Build** — Handles local images via `podman build` if `build:` contexts are defined, validates existing local images, or pulls missing ones from the registry.
5. **Deploy** — Relocates the finalized files to the systemd configuration directory, registers the metadata in the centralized state file, and triggers the unit starts via D-Bus.

### Dry Run Mode (`--dry-run`)

When `comquad up --dry-run` is used, the pipeline runs steps 1–3 into a private temporary directory instead of the real systemd target. After cooking, `printDryRun` reads each generated file and prints:

- The **target path** it *would* be written to
- The full **file content** of each quadlet
- **Image actions** (build/pull) that *would* be taken per service, based on the pull strategy and whether images exist locally

Steps 4–5 are skipped entirely: no files are written to the systemd directory, no state is registered, and no units are started. The temporary preview directory is cleaned up automatically.

## 📦 Project Directory Structure

The codebase is organized cleanly into domains matching the execution lifecycle steps:

```text
cmd/comquad/           # CLI entry point managed by Cobra commands
internal/build/        # Image building and registry pulling routines
internal/cooker/       # Post-processes quadlet files (renaming, reference rewriting)
internal/deploy/       # Systemd D-Bus communication, target directories, state tracking,
                       # and the SystemdClient / StateStore interfaces used for testing
internal/logger/       # Colorized verbose logging utility
internal/orchestrator/ # The engine that wires all packages together to drive the up/down lifecycle
internal/preprocess/   # Pre-parser to normalize raw compose.yaml files
internal/transpile/    # Wrapper executing the podlet binary

```

## 📊 Ps Command

The `ps` command shows container runtime status in `docker compose ps` style. It queries Podman for container data and merges it with systemd D-Bus unit state.

**Data sources:**

1. **Podman** — Runs `podman ps --filter "label=com.comquad.managed=true" --filter "label=com.comquad.project=<name>" --format json` (or `-a` flag for exited containers). Parses JSON to extract container name, image, command, state, status, ports, networks, mounts, exit code, and timestamps.
2. **D-Bus** — Queries `ListAllUnits()` and builds a map keyed by unit name. Merges `ActiveState` and `SubState` into each container record.

**Output format:**

```
NAME                 IMAGE                          COMMAND                   SERVICE      CREATED              STATUS               PORTS
--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
my_nginx             docker.io/library/nginx:alpine nginx -g daemon off;      nginx        2 minutes ago        Up 2 minutes         0.0.0.0:2080->80/tcp
2nginx               docker.io/library/nginx:alpine nginx -g daemon off;      nginx2       2 minutes ago        Up 2 minutes         0.0.0.0:2081->80/tcp
```

Columns are auto-width based on content. Exited containers show `Exited (<code>) <time>` in the status column. Dead containers show `Dead`. Ports are formatted as `host_ip:host_port->container_port/protocol`. Created time uses relative format (`just now`, `5m ago`, `2d ago`, or `Jan 02 2006` for older entries).

**Flags:**

* `-a, --all` — Include exited/dead containers (uses `podman ps -a`)

**Testability:** `Orchestrator.listContainers` is an injectable function field (`func(projectName string, all bool) ([]ContainerInfo, error)`), allowing tests to provide canned container data without a live Podman daemon.

## 👁️ View Command

The `view` command provides two modes of inspection:

**Project view** (no service argument): queries systemd D-Bus for all units belonging to a project, computes aggregate health status, and displays a table of `UNIT`, `ACTIVE`, and `SUB` states. Status is `healthy` when all units are active, `down` when none are active, and `degraded` when only some are active.

**Unit file view** (with service argument): resolves the quadlet file using five matching patterns (`web` → `cq-myapp-web.container`, `cq-myapp-web`, `cq-myapp-web.service`, `cq-myapp-web.container`, or `myapp-web`), reads the file, and prints its contents.

Unit resolution iterates over `state.Files` from `projects.json`, checking each pattern in order until a match is found.

## ✏️ Edit Command

The `edit` command provides two modes of file editing:

**Project edit** (no service argument): resolves all `.container`, `.network`, and `.volume` quadlet files for a project and opens them in `$EDITOR` (falls back to `vi`). `$EDITOR` is split on whitespace, so values like `"vim -o"` or `"code --wait"` work correctly. After the editor exits, comquad compares file contents and auto-reloads systemd, restarting any changed container units.

**Unit file edit** (with service argument): resolves a single quadlet file using the same matching patterns as `view`, opens it in the editor, and reloads systemd if changes were detected.

The `--no-reload` flag opens files without triggering a systemd daemon reload or unit restart.

Unit resolution shares the same matching logic as `view`, using `MatchContainer` and `MatchNetworkOrVolume` helpers that iterate over `state.Files` from `projects.json`.

## 📋 Logs Command

The `logs` command queries systemd D-Bus to determine each unit's state and filters output accordingly:

**Running units** (`ActiveState == "active"`): retrieves the `InvocationID` property via D-Bus `GetUnitProperties` and passes it to `journalctl --invocation=<hex>`, showing only logs from the current invocation.

**Stopped / failed units**: no filter is applied, showing full historical logs.

Service name matching uses the same multi-pattern logic as `view` and `edit`: exact file name, name without extension, name with `.service` suffix, short name (after stripping `cq-<project>-` prefix), internal Podman name (after stripping `cq-` prefix), or `ContainerName=` directive from the unit file. `MatchContainers` returns all matching files per argument, allowing a single arg like `web` to match multiple services.

### Flags

- `--tail <N>` — Limit output to the last N lines
- `--since <time>` — Show logs since a specific time (e.g. `10m`, `2024-01-01 12:00:00`)
- `--output <format>` — Override journalctl output format (default: `cat` to strip raw systemd metadata like boot ID and machine ID)

When querying multiple units, each log line is prefixed with `[<unit-name>]` to identify its source. Lines already starting with `--` (journalctl's timestamp separators) are passed through unmodified. All empty lines (journalctl's `--output cat` entry separators) are stripped in every code path.

## 🔄 Lifecycle Commands (Start, Stop, Restart)

The `start`, `stop`, and `restart` commands manage the runtime state of deployed projects without touching quadlet files or triggering daemon-reload. They operate directly via D-Bus.

**Service resolution:** All three commands accept optional `[service ...]` positional arguments. When provided, they use `MatchContainers` to resolve service names to unit names. When omitted, all `.container` files from the project's state are started/stopped/restarted.

**Start** — Iterates over resolved unit names and calls `StartUnit` via D-Bus. Reports per-unit status messages.

**Stop** — Iterates over resolved unit names and calls `StopUnit` via D-Bus, then verifies all units are no longer active (same verification as `down`).

**Restart** — Iterates over resolved unit names and calls `RestartUnit` via D-Bus, which tears down and recreates the unit cleanly.

All three commands require the project to exist in `projects.json` state. They share a `resolveUnits()` helper that looks up the project state, matches service names, and deduplicates results.

## 🗑️ Down Command

The `down` command performs a complete teardown of a deployed project in six steps:

1. **Stop units** — Stops all container units via systemd D-Bus `StopUnit`, then verifies all units are no longer active.
2. **Remove quadlet files** — Deletes all `.container`, `.network`, and `.volume` files from the systemd target directory.
3. **Reload daemon** — Triggers `daemon-reload` via D-Bus so systemd forgets the removed units and releases its references to networks and volumes.
4. **Remove networks** — Lists all Podman networks with label `com.comquad.managed=true` and project label matching the current project, then removes them via `podman network rm`.
5. **Remove volumes (opt-in)** — When the `-d, --delete-volumes` flag is provided, lists all Podman volumes with label `com.comquad.managed=true` and project label matching the current project, then removes them via `podman volume rm`. Volumes are opt-in because they may contain persistent data.
6. **Unregister project** — Removes the project entry from `projects.json` state file.

**Usage:**

```bash
comquad down          # stops containers, removes networks, removes quadlet files
comquad down -d       # also removes Podman volumes
```

## 🐳 Exec Command

The `exec` command runs a command inside a running container via `podman exec`. It requires a single service argument and allocates a TTY by default (like `docker compose exec`).

**Service resolution:** Uses `MatchContainers` to resolve the service name to a container quadlet file. The container name is derived from the base filename by stripping the `cq-` prefix and `.container` suffix (e.g. `cq-myapp-web.container` → `myapp-web`). If the service matches multiple containers, an error is returned listing the ambiguous matches.

**Flags:** `-u/--user` sets the user inside the container, `-t/--tty` controls TTY allocation (default `true`). The command is passed directly to `podman exec`, which handles `--` flag separation.

## 🔄 Regenerate Command

The `regenerate` command restores the state file by scanning Podman for managed resources. It is the foundation of comquad's self-healing state management.

**Discovery pipeline:**

1. Queries Podman for all containers with label `com.comquad.managed=true`
2. Queries Podman for all networks with label `com.comquad.managed=true`
3. Queries Podman for all volumes with label `com.comquad.managed=true`
4. Groups all resources by their `com.comquad.project` label value
5. Resolves quadlet files in the systemd target directory matching `cq-<project>-*.container`, `*.network`, `*.volume`
6. Writes the reconstructed state to `projects.json`

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
  "files": ["/path/to/cq-myproject-web.container"],
  "resources": {
    "containers": ["cq-myproject-web"],
    "networks": ["cq-myproject-default-network"],
    "volumes": []
  }
}
```

* **project_name** — The Comquad project name
* **source_path** — Path to the compose file directory (empty when restored via `regenerate`)
* **files** — List of quadlet file paths in the systemd target directory
* **resources** — Podman resources discovered via labels (`containers`, `networks`, `volumes`). Populated by `regenerate`. When `up` re-registers an existing project, any previously stored resource info is preserved.

The state file is written atomically: comquad writes to a `.tmp` file in the same directory and renames it, so a crash mid-write never produces a corrupt state file.

### Target Systemd Directories

Quadlet configurations are copied into paths dictated by your execution context:

* **Rootless Mode (Default user):** `~/.config/containers/systemd`
* **Root Mode (UID 0 / sudo):** `/etc/containers/systemd`

---

## 📋 Compose Format Implementation Details

`comquad` accepts standard Docker Compose v3 files supporting `services`, `networks`, and `volumes`.

The following fields accept both map (`KEY: value`) and list (`- KEY=value`) formats:

* `environment` — service environment variables
* `labels` — service, network, and volume labels
* `build.args` — build arguments

The following compose service fields are **handled by comquad** (explicitly processed or auto-injected):

* `container_name` — auto-generated if missing (`<project>-<service>`); also registered as a second `NetworkAlias=` for DNS resolution
* `image` — normalized to full registry path (`docker.io/library/`)
* `build` — local build context (string or map)
* `ports` — published host ports (offset in rootless mode)
* `volumes` — bind mounts and named volumes (relative paths resolved)
* `networks` — network attachments (auto-attached to `cq-default` if none defined)
* `entrypoint` — container entrypoint
* `command` — container command
* `expose` — ports exposed to linked services
* `deploy` — deploy-time configuration
* `environment` — environment variables (map or list format)
* `labels` — service, network, and volume labels (map or list format)

The following compose service fields are **not yet handled** by comquad but are **passed through unchanged** to `podlet`: `depends_on`, `restart`, `working_dir`, `user`, `healthcheck`, `cap_add`/`cap_drop`, `tmpfs`, `read_only`, `extra_hosts`, `dns`, `hostname`, `privileged`, `mem_limit`, `cpus`, `volumes_from`, `links`, `tty`, `stdin_open`, `security_opt`, `shm_size`. Unknown top-level keys (e.g. `version`, `x-` extensions, `secrets`, `configs`) are also preserved.

### Automatic Behaviors & Opinionated Transforms

To ensure the transition to Quadlets is frictionless, the internal engine enforces several rules:

* Relative volume host paths are automatically fully-qualified to absolute paths.
* When SELinux is detected (via `/sys/fs/selinux/enforce`), all `Volume=` directives in generated `.container` files get `,z` appended to mount options (`:ro` → `:ro,z`, `:rw` → `:rw,z`, no option → `:z`). Idempotent — skips if `:z` or `:Z` already present.
* A default bridge network (`cq-default`) is implicitly injected only when the compose file defines no networks at all. Services without an explicit `networks:` key are auto-attached to `cq-default` only when that network was injected — preventing dangling network references when user-defined networks exist.
* Generated containers follow a strict naming blueprint: `<project>-<service>`.
* `NetworkAlias=` is injected into every `.container` file so services can resolve each other by service name and `ContainerName=` value within compose networks.
* An identifying label (`com.comquad.project`) is attached to all generated units.
* A `com.comquad.managed` label is attached to all files to indicate comquad management.
* Unprefixed public images default seamlessly to standard Docker Hub (`docker.io/library/`).
* Services marked with local `build:` blocks bypass image registry name validation.
* In rootless mode, privileged ports (< 1024) are automatically offset by `ROOTLESS_PORT_OFFSET` (default 2000). Internal port conflicts within a project are resolved by incrementing.

### Local Build Rules

When a service contains a `build:` block, `comquad` builds it on-the-fly and tags it as `<project>-<service>:latest` before creating the Quadlet units.

Standard shorthand formats and extended structural contexts are both supported:

```yaml
services:
  web:
    build:
      context: ./apps/web          # Build context directory (default: .)
      dockerfile: Dockerfile.prod  # Custom Dockerfile name (default: Dockerfile)
      target: production           # Build target stage
      args:                        # Build arguments (map or list format)
        VERSION: "1.0"
      # args:                        # Also supported:
      #   - VERSION=1.0
```

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

## 🧪 Testing Architecture

The orchestrator package was historically untestable because it constructed `SystemdManager` and `StateManager` directly inside every method. Two interfaces in `internal/deploy/interfaces.go` solve this:

- **`SystemdClient`** — all nine D-Bus methods used by the orchestrator (`StartUnit`, `StopUnit`, `RestartUnit`, `ReloadDaemon`, `WaitForUnit`, `ListUnitsByNames`, `ListAllUnits`, `GetInvocationID`, `Close`). The concrete `SystemdManager` satisfies this interface.
- **`StateStore`** — all state operations used by the orchestrator (`GetProject`, `GetStateFilePath`, `ListProjects`, `RegisterProject`, `UnregisterProject`, `Save`). The concrete `StateManager` satisfies this interface. `GetProject` replaces direct `Projects[name]` map access, making the interface satisfiable without exposing the map.

`Orchestrator` holds three factory fields instead of calling the constructors directly:

```go
newState       func() (deploy.StateStore, error)
newSystemd     func() (deploy.SystemdClient, error)
listContainers func(projectName string, all bool) ([]ContainerInfo, error)
```

`NewOrchestrator` wires in the real implementations. Tests override these fields with in-memory fakes (`mockStateStore`, `mockSystemdClient`) that record calls and return canned responses, enabling full unit-test coverage without a live D-Bus or Podman daemon.

The `transpile` package is tested via a fake `podlet` shell script placed on a temp PATH entry, exercising the stdin pipe, argument passing, and error paths without requiring the real binary.

## 📋 Follow Logs on Deploy

When `comquad up -f` is used, after successfully deploying all units the CLI captures the current timestamp and streams all journal logs for every project unit (containers, networks, and volumes) from that point onward. This emulates the default `docker compose up` behavior (without `-d`), keeping the terminal attached to live output until interrupted with Ctrl+C.

The deployment timestamp is captured after image handling completes but before `daemon-reload` and unit starts, ensuring no startup logs are missed.

The follow mode uses the same `cat` default output format and supports the `-n/--tail` and `--output` flags via the `FollowLogs()` function.

## 📊 Output Levels

comquad has three output modes, controlled by flags on the root command:

| Mode | Flag | What's shown |
|---|---|---|
| Normal (default) | *(none)* | Operational messages: unit starts/stops, deploy success, errors |
| Verbose | `-v` / `--verbose` | All of the above plus every pipeline transformation |
| Quiet | `-q` / `--quiet` | Errors only (stderr). All other output is suppressed. Useful in scripts. |

`--quiet` takes precedence over `--verbose`. `logger.Error(...)` always writes to stderr regardless of either flag.

The logger lives in `internal/logger/` and exposes three tiers:
- **`logger.Print(msg)`** — normal operational output, suppressed by `--quiet`
- **`logger.Action/Info/Success/Warn(msg)`** — verbose-only, also suppressed by `--quiet`
- **`logger.Error(msg)`** — always to stderr, never suppressed

Colors use ANSI codes (green=success, cyan=info, yellow=warning, red=error, blue=action) and are disabled when `NO_COLOR` is set.

When `-v` / `--verbose` is enabled, comquad additionally logs every transformation applied during the deployment pipeline:

**Preprocess stage** logs:
- `Injected container_name: <name>` — when a container name was auto-generated
- `Normalized image: <original> → <normalized>` — image name normalization to full registry path
- `Normalized volume path: <relative> → <absolute>` — relative volume path resolution
- `Created default network: cq-default` — when a default bridge network was injected
- `Auto-attached '<service>' to network 'cq-default'` — services auto-attached to default network

**Cook stage** logs:
- `Renamed <old> → <new>` — file renaming with `cq-<project>-` prefix
- `Rewrote cross-unit references in <file>` — reference rewriting for Network=/Volume=/Pod= and [Unit] section (After=, Requires=, etc.) directives
- `Added AutoUpdate=registry to <file>` — systemd auto-update optimization
- `Added [Install] section to <file>` — systemd install section injection
- `Added NetworkAlias=<name> to <file>` — DNS resolution for compose networks
- `Added labels: Label=com.comquad.project=<name>, Label=com.comquad.managed=true` — label injection
- `Offset port: PublishPort=<original> → PublishPort=<offset>` — rootless port offsetting

**Build stage** logs:
- `Building image for service <name>: <tag>` — local image builds
- `Built image: <tag>` — build completion
- `Pulling image: <image>` — image pulls
- `Handled image: <image>` — image handling completion

**Down stage** logs (verbose):
- `Removing network: <name>` — network removal
- `Removing volume: <name>` — volume removal

Network and volume removal errors are always printed to stderr regardless of verbose mode, so cleanup failures are never silently dropped.

**Error output:** `logger.Error(...)` always writes to stderr regardless of the verbose setting. Only informational, success, warning, and action messages are gated behind `-v`.

