# comquad (Compose + Quadlet 🍊)

`comquad` is a Docker-compose-like CLI for Podman Quadlets, backed by systemd.

It lets you define your services in a standard `compose.yaml` file and deploy them as individual systemd units using Podman's Quadlet technology. Instead of running its own orchestrator, `comquad` prepares the quadlet files and delegates lifecycle management entirely to systemd.

---

## 🚧 Project Status: Infra-Built Utility

I am an infrastructure engineer, not a full-time software developer. I built **Comquad** to solve a specific problem for my own workflow.

* **Philosophy:** This tool is intentionally small, simple, and transparent. It is not trying to become Kubernetes.
* **Contributions:** I am currently not accepting complex feature pull requests because I do not have the bandwidth or Go expertise to maintain them.
* **Bugs:** Feel free to open issues if a specific Docker Compose file breaks, but fixes will happen on a "best effort" timeline.

---

## 🛠️ Requirements & Installation

### Requirements

* **Podman** (for `podman pull`)
* **podlet** (for transpiling `compose` yaml into quadlet files)
* **systemd** with quadlet support
* Go 1.23+ (if building from source)

### Installation

```bash
# Build from source
go build -o comquad ./cmd/comquad/
sudo cp comquad /usr/local/bin/

# Or install directly via Go
go install comquad/cmd/comquad@latest

```

---

## ⚙️ Core Usage Workflow

### 1. Deploying a Project (`up`)

From a directory containing your `compose.yaml`:

```bash
comquad up

```

* **Follow logs:** `comquad up -f` streams journal logs from the deployment timestamp.
* **Force rebuild:** `comquad up --build` forces an image rebuild.
* **Image Pull Control:** `comquad up --pull [always|missing|never]` *(default: missing)*.
* **Override name:** `comquad up -n my-service` overrides the default project name.

### 2. Monitoring & Lifecycle (`ps`, `start`, `stop`, `logs`)

```bash
# View container status (Docker Compose style)
comquad ps
comquad ps -a  # Includes exited containers

# Control services
comquad start [service ...]
comquad stop [service ...]
comquad restart [service ...]

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
comquad down -d                      # Also removes Podman volumes

```

---

## 🔍 Advanced Features & Inspecting State

### Dry Run & Verbose Preview

Before committing changes to systemd, you can preview exactly what `comquad` will do:

```bash
# Preview generated files without writing them
comquad up --dry-run

# Show every transformation (port offsets, path normalizations, etc.)
comquad up -v

```

### Direct Unit Editing & Viewing

You can view or edit the underlying systemd quadlet files on the fly:

```bash
# View/Cat the unit files
comquad view myapp web       # Cat the cq-myapp-web.container file
comquad view                 # View all units for the current project

# Edit unit files directly (automatically triggers systemd daemon-reload)
comquad edit myapp web
comquad edit --no-reload     # Open files without auto-reloading systemd

```

### Self-Healing & Repair

If your local state gets out of sync, `comquad` can rebuild its tracking from Podman labels:

```bash
comquad regenerate --force           # Reconstruct state file from live labels
comquad regenerate --force --dry-run # Preview what would be reconstructed
comquad check                        # Run a quick prerequisites sanity check

```

---

## 🏗️ Architecture & Automatic Behaviors

`comquad` uses a schema-less YAML model to preserve all compose file fields through its preprocessing pipeline. Any field not explicitly handled (like `depends_on`, `healthcheck`, or `x-` extensions) is passed through completely unchanged to `podlet`.

For a deep dive into how `comquad` processes compose files, manages state, and maps directories, check out the [Architecture Guide](https://www.google.com/search?q=./ARCHITECTURE.md).

### Behind-the-Scenes Automations:

* **Path Fixing:** Relative volume host paths are automatically fully qualified to absolute paths.
* **SELinux Smart Patching:** When SELinux is active on the host, all `Volume=` directives automatically get `,z` or `:z` flags appended safely and idempotently.
* **Implicit Networks:** A default bridge network (`cq-default`) is injected if your compose file defines no networks.
* **Service Discovery:** `NetworkAlias=` and unique `<project>-<service>` blueprints are injected into every `.container` file so systemd services can resolve each other.
* **Rootless Port Offsetting:** In rootless mode, privileged ports (< 1024) are automatically shifted by `ROOTLESS_PORT_OFFSET` (default: `2000`) to prevent deployment failures.

---

## 📄 License

MIT
