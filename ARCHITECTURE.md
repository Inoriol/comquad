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

