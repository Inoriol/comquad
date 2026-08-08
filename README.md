# comquad (Compose + Quadlet 🍊)

`comquad` is a Docker-compose-like CLI for Podman Quadlets, backed by systemd.

It lets you define your services in a standard `compose.yaml` file and deploy them as individual systemd units using Podman's Quadlet technology. Instead of running its own orchestrator, `comquad` prepares the quadlet files and delegates lifecycle management entirely to systemd.

---

## 🚧 Project Status: Infra-Built Utility

I am an infrastructure engineer, not a full-time software developer. I built **Comquad** to solve a specific problem for my own workflow.

* **Philosophy:** This tool is intentionally small, simple, and transparent. It is not trying to become Kubernetes. It's not trying to become podman compose 2.0 either.
* **Contributions:** I am currently not accepting complex feature pull requests because I do not have the bandwidth or Go expertise to maintain them. But I'm very open to suggestions.
* **Bugs:** Feel free to open issues if a specific Docker Compose file breaks, but fixes will happen on a "best effort" timeline.

---

## 🛠️ Requirements & Installation

### Requirements

* **Podman 4.4+** (quadlet support)
* **podlet** (for transpiling `compose` yaml into quadlet files)
* **systemd** with quadlet support
* Go 1.23+ (if building from source)

### Installation

```bash
# Build from source (with version)
go build -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o comquad ./cmd/comquad/
sudo cp comquad /usr/local/bin/

# Or install directly via Go
go install github.com/Inoriol/comquad/cmd/comquad@latest

# Verify
comquad --version
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `EDITOR` | auto-detected | Editor for `comquad edit`. Falls back to `editor`, `nano`, `vim`, then `vi`. |
| `NO_COLOR` | *(unset)* | Set to any value to disable ANSI color output. |
| `ROOTLESS_PORT_OFFSET` | `2000` | In rootless mode, privileged ports (< 1024) are offset by this value. |
| `XDG_DATA_HOME` | `~/.local/share` | Base directory for `comquad/projects.json` state file. |

---

## ⚙️ Core Usage Workflow

### 1. Deploying a Project (`up`)

From a directory containing your `compose.yaml`:

```bash
comquad up

```

* **Follow logs:** `comquad up -f` streams journal logs from the deployment timestamp.
* **Image Pull Control:** `comquad up --pull [always|missing|never]` *(default: missing)*.
* **Override name:** `comquad up -n my-service` overrides the default project name.
* **Progress indication:** Pipeline stages are reported during deployment (`--verbose`/`-v` for full detail).

### 2. Monitoring & Lifecycle (`ps`, `start`, `stop`, `logs`)

```bash
# View container status (Docker Compose style)
comquad ps
comquad ps -a  # Includes exited containers

# Control services
comquad start [service ...]
comquad start --dry-run        # Preview which units would be started
comquad stop [service ...]
comquad stop --dry-run         # Preview which units would be stopped
comquad restart [service ...]
comquad restart --dry-run      # Preview which units would be restarted

# Stream logs (auto-sorted chronologically across units)
comquad logs                 # All services (one-shot)
comquad logs -f              # All services (follow)
comquad logs web             # Single service
comquad logs --tail 50       # Last 50 lines
comquad logs --since 10m     # Last 10 minutes

```

### 3. Interacting & Tearing Down (`exec`, `down`)

```bash
# Run commands inside containers
comquad exec web ls /app
comquad exec web sh                  # Interactive TTY shell
comquad exec -u root web bash        # Run as root

# Tear down the project
comquad down
comquad down -y                  # Skip confirmation prompt
comquad down -d                  # Also removes Podman volumes
comquad down --dry-run           # Preview what would be removed

```

---

## 🔍 Advanced Features & Inspecting State

### Dry Run & Verbose Preview

Before committing changes to systemd, you can preview exactly what `comquad` will do:

```bash
# Preview generated files without writing them
comquad up --dry-run

# Preview lifecycle actions without affecting running units
comquad start --dry-run
comquad stop --dry-run
comquad restart --dry-run
comquad down --dry-run

# Show every transformation (port offsets, path normalizations, etc.)
comquad up -v
comquad down -v     # Also works with all subcommands
comquad ps -v

```

### Direct Unit Editing & Viewing

You can view or edit the underlying systemd quadlet files on the fly:

```bash
# View the project overview with resource relationships
comquad view                 # Also accessible via `comquad overview`
comquad view myapp web       # Shows the cq-myapp-web.container file content
comquad view db.image        # View a specific .image quadlet file

# Edit unit files directly (automatically triggers systemd daemon-reload)
comquad edit myapp web
comquad edit --no-reload     # Open files without auto-reloading systemd

```

The `view` command provides a clean relational display showing each container's image, attached networks, and volumes, plus a separate table of all managed resources (images, networks, volumes) with copy-pasteable names.

### Self-Healing & Repair

If your local state gets out of sync, `comquad` can rebuild its tracking from Podman labels:

```bash
comquad regenerate --force           # Reconstruct state file from live labels
comquad regenerate --force --dry-run # Preview what would be reconstructed
comquad check                        # Check prerequisites (tools, podman >= 4.4, D-Bus, target dir)

```

### Managing Projects

```bash
# List all deployed projects (also accessible as `comquad ls`)
comquad list

# Shell completion generation
comquad completion bash              # Generate for bash
comquad completion zsh               # Generate for zsh
comquad completion fish              # Generate for fish

# Help with examples
comquad up --help                    # Each command shows usage examples
```

### Getting Help

All commands have built-in examples — just append `--help`:

```bash
comquad up --help      # Deploy examples
comquad logs --help    # Logging examples with --since/--tail
comquad exec --help    # Container exec examples
```

---

## 🏗️ Architecture & Automatic Behaviors

`comquad` uses a schema-less YAML model to preserve all compose file fields through its preprocessing pipeline. `pull_policy` and `platform` fields are intercepted and moved into dedicated `.image` quadlet files during the graft step. Most unhandled fields (like `depends_on`, `healthcheck`, or `x-` extensions) are passed through unchanged to `podlet`. However, `build:` blocks are currently explicitly rejected — build support is planned for a future release.

For a deep dive into how `comquad` processes compose files, manages state, and maps directories, check out the [Architecture Guide](./ARCHITECTURE.md).

### Behind-the-Scenes Automations:

* **Path Fixing:** Relative volume host paths are automatically fully qualified to absolute paths.
* **SELinux Smart Patching:** When SELinux is active on the host, all `Volume=` directives automatically get `,z` or `:z` flags appended safely and idempotently.
* **Implicit Networks:** A default bridge network (`cq-default`) is injected if your compose file defines no networks.
* **Image Quadlet Generation:** Every container gets a companion `.image` quadlet file. Compose `image`, `pull_policy`, and `platform` fields are extracted into dedicated image units so systemd can manage image pulls separately (enables `podman auto-update`).
* **Service Discovery:** `NetworkAlias=` and unique `<project>-<service>` blueprints are injected into every `.container` file so systemd services can resolve each other.
* **Rootless Port Offsetting:** In rootless mode, privileged ports (< 1024) are automatically shifted by `ROOTLESS_PORT_OFFSET` (default: `2000`) to prevent deployment failures.

---

## 📄 License

MIT
