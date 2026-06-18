# Architecture & Technical Design

This document details the internal design, component mapping, and execution lifecycle of **comquad**. Read this to understand how your Docker Compose components translate safely into native systemd Quadlets.

## 🔄 Execution Lifecycle

When you run `comquad up`, the engine moves your configuration through a five-step pipeline:

1. **Preprocess** — Normalizes your `compose` yaml (resolves relative to absolute paths, sets default networks, injects project labels).
2. **Transpile** — Executes the `podlet` binary under the hood to convert the compose YAML configuration into `.container`, `.network`, and `.volume` quadlet files.
3. **Cook** — Post-processes the raw quadlet outputs. This stage prefixes files with `cq-<project>`, rewrites cross-unit references so services can communicate, injects `com.comquad.managed` and `com.comquad.project` labels on all files, and applies rootless port offsets where needed.
4. **Build** — Handles local images via `podman build` if `build:` contexts are defined, validates existing local images, or pulls missing ones from the registry.
5. **Deploy** — Relocates the finalized files to the systemd configuration directory, registers the metadata in the centralized state file, and triggers the unit starts via D-Bus.

## 📦 Project Directory Structure

The codebase is organized cleanly into domains matching the execution lifecycle steps:

```text
cmd/comquad/           # CLI entry point managed by Cobra commands
internal/build/        # Image building and registry pulling routines
internal/cooker/       # Post-processes quadlet files (renaming, reference rewriting)
internal/deploy/       # Systemd D-Bus communication, target directories, and state tracking
internal/orchestrator/ # The engine that wires all packages together to drive the up/down lifecycle
internal/preprocess/   # Pre-parser to normalize raw compose.yaml files
internal/transpile/    # Wrapper executing the podlet binary

```

## 👁️ View Command

The `view` command provides two modes of inspection:

**Project view** (no service argument): queries systemd D-Bus for all units belonging to a project, computes aggregate health status (`healthy` / `degraded` / `down`), and displays a table of `UNIT`, `ACTIVE`, and `SUB` states.

**Unit file view** (with service argument): resolves the quadlet file using five matching patterns (`web` → `cq-myapp-web.container`, `cq-myapp-web`, `cq-myapp-web.service`, `cq-myapp-web.container`, or `myapp-web`), reads the file, and prints its contents.

Unit resolution iterates over `state.Files` from `projects.json`, checking each pattern in order until a match is found.

## ✏️ Edit Command

The `edit` command provides two modes of file editing:

**Project edit** (no service argument): resolves all `.container`, `.network`, and `.volume` quadlet files for a project and opens them in `$EDITOR` (falls back to `vi`). After the editor exits, comquad compares file contents and auto-reloads systemd, restarting any changed container units.

**Unit file edit** (with service argument): resolves a single quadlet file using the same matching patterns as `view`, opens it in the editor, and reloads systemd if changes were detected.

The `--no-reload` flag opens files without triggering a systemd daemon reload or unit restart.

Unit resolution shares the same matching logic as `view`, using `MatchContainer` and `MatchNetworkOrVolume` helpers that iterate over `state.Files` from `projects.json`.

## 📋 Logs Command

The `logs` command queries systemd D-Bus to determine each unit's state and filters output accordingly:

**Running units** (`ActiveState == "active"`): retrieves the `InvocationID` property via D-Bus `GetUnitProperties` and passes it to `journalctl --invocation=<hex>`, showing only logs from the current invocation.

**Stopped / failed units**: no filter is applied, showing full historical logs.

Service name matching uses the same multi-pattern logic as `view` and `edit`: exact file name, name without extension, name with `.service` suffix, short name (after stripping `cq-<project>-` prefix), or internal Podman name (after stripping `cq-` prefix). `MatchContainers` returns all matching files per argument, allowing a single arg like `web` to match multiple services.

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
2. **Remove networks** — Lists all Podman networks matching the `cq-<project>-` or `*-<project>` prefix and removes them via `podman network rm`.
3. **Remove volumes (opt-in)** — When the `-v, --volumes` flag is provided, lists all Podman volumes matching the same prefix pattern and removes them via `podman volume rm`. Volumes are opt-in because they may contain persistent data.
4. **Remove quadlet files** — Deletes all `.container`, `.network`, and `.volume` files from the systemd target directory.
5. **Reload daemon** — Triggers `daemon-reload` via D-Bus so systemd forgets the removed units.
6. **Unregister project** — Removes the project entry from `projects.json` state file.

**Usage:**

```bash
comquad down          # stops containers, removes networks, removes quadlet files
comquad down -v       # also removes Podman volumes
```

## 🐳 Exec Command

The `exec` command runs a command inside a running container via `podman exec`. It requires a single service argument and allocates a TTY by default (like `docker compose exec`).

**Service resolution:** Uses `MatchContainers` to resolve the service name to a container quadlet file. The container name is derived by stripping `cq-` prefix and `.container` suffix (e.g. `cq-myapp-web.container` → `myapp-web`). If the service matches multiple containers, an error is returned listing the ambiguous matches.

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
* `--dry-run` — Preview what would be regenerated without writing the state file

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
* **resources** — Podman resources discovered via labels (`containers`, `networks`, `volumes`). Populated by `regenerate` and kept in sync during `up`/`down`.

### Target Systemd Directories

Quadlet configurations are copied into paths dictated by your execution context:

* **Rootless Mode (Default user):** `~/.config/containers/systemd`
* **Root Mode (UID 0 / sudo):** `/etc/containers/systemd`

---

## 📋 Compose Format Implementation Details

`comquad` accepts standard Docker Compose v3 files supporting `services`, `networks`, and `volumes`.

### Automatic Behaviors & Opinionated Transforms

To ensure the transition to Quadlets is frictionless, the internal engine enforces several rules:

* Relative volume host paths are automatically fully-qualified to absolute paths.
* A default bridge network is implicitly injected if your file omits a network definition.
* Every service container without an assigned network is auto-attached to that default project network.
* Generated containers follow a strict naming blueprint: `<project>-<service>`.
* An identifying label (`com.comquad.project`) is attached to all generated units.
* A `com.comquad.managed` label is attached to all files to indicate comquad management.
* Unprefixed public images default seamlessly to standard Docker Hub (`docker.io/library/`).
* Services marked with local `build:` blocks bypass image registry name validation.
* In rootless mode, privileged ports (< 1024) are automatically offset by `COMQUAD_PORT_OFFSET` (default 2000). Internal port conflicts within a project are resolved by incrementing.

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
      args:                        # Build arguments
        VERSION: "1.0"

```

### Quadlet Feature Injections

You can inject native Quadlet behaviors via labels. For example, to prevent auto-updates on a specific container image, set `comquad-no-autoupdate`:

```yaml
services:
  web:
    image: nginx
    labels:
      comquad-no-autoupdate: "true"

```

## 📋 Follow Logs on Deploy

When `comquad up -f` is used, after successfully deploying all units the CLI captures the current timestamp and streams all journal logs for every project unit (containers, networks, and volumes) from that point onward. This emulates the default `docker compose up` behavior (without `-d`), keeping the terminal attached to live output until interrupted with Ctrl+C.

The deployment timestamp is captured after image handling completes but before `daemon-reload` and unit starts, ensuring no startup logs are missed.

