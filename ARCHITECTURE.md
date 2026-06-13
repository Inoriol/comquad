# Architecture & Technical Design

This document details the internal design, component mapping, and execution lifecycle of **comquad**. Read this to understand how your Docker Compose components translate safely into native systemd Quadlets.

## 🔄 Execution Lifecycle

When you run `comquad up`, the engine moves your configuration through a five-step pipeline:

1. **Preprocess** — Normalizes your `compose` yaml (resolves relative to absolute paths, sets default networks, injects project labels).
2. **Transpile** — Executes the `podlet` binary under the hood to convert the compose YAML configuration into `.container`, `.network`, and `.volume` quadlet files.
3. **Cook** — Post-processes the raw quadlet outputs. This stage prefixes files with `cq-<project>`, rewrites cross-unit references so services can communicate, and applies rootless port offsets where needed.
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

**Unit file view** (with service argument): resolves the quadlet file using four matching patterns (`web` → `cq-myapp-web.container`, `cq-myapp-web`, `cq-myapp-web.service`, or `cq-myapp-web.container`), reads the file, and prints its contents.

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

Service name matching uses the same multi-pattern logic as `view` and `edit`: exact file name, name without extension, name with `.service` suffix, or short name (after stripping `cq-<project>-` prefix). `MatchContainers` returns all matching files per argument, allowing a single arg like `web` to match multiple services.

## 🔄 Lifecycle Commands (Start, Stop, Restart)

The `start`, `stop`, and `restart` commands manage the runtime state of deployed projects without touching quadlet files or triggering daemon-reload. They operate directly via D-Bus.

**Service resolution:** All three commands accept optional `[service ...]` positional arguments. When provided, they use `MatchContainers` to resolve service names to unit names. When omitted, all `.container` files from the project's state are started/stopped/restarted.

**Start** — Iterates over resolved unit names and calls `StartUnit` via D-Bus. Reports per-unit status messages.

**Stop** — Iterates over resolved unit names and calls `StopUnit` via D-Bus, then verifies all units are no longer active (same verification as `down`).

**Restart** — Iterates over resolved unit names and calls `RestartUnit` via D-Bus, which tears down and recreates the unit cleanly.

All three commands require the project to exist in `projects.json` state. They share a `resolveUnits()` helper that looks up the project state, matches service names, and deduplicates results.

## 💾 State & File System Management

### State File Location

`comquad` keeps track of active, managed projects in a light JSON database.

* **Default:** `~/.local/share/comquad/projects.json`
* **Overridden by:** `$XDG_DATA_HOME/comquad/projects.json`

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

