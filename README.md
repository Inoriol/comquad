# comquad

ComQuad (Compose + Quadlet 🍊) is a Docker-compose-like CLI for Podman Quadlets, backed by systemd.

`comquad` lets you define your services in a `compose` yaml file and deploy them as individual systemd units using Podman's Quadlet technology. Instead of running its own orchestrator, comquad prepares the quadlet files and delegates lifecycle management entirely to systemd.

## 🚧 Project Status: Infra-Built Utility

I am an infrastructure engineer, not a full-time software developer. I built **Comquad** to solve a specific problem for my own workflow.

- **Contributions:** I am currently not accepting complex feature pull requests because I do not have the bandwidth or Go expertise to maintain them.
- **Bugs:** Feel free to open issues if a specific Docker Compose file breaks, but fixes will happen on a "best effort" timeline.
- **Philosophy:** This tool is intentionally small, simple, and transparent. It is not trying to become Kubernetes.

## 🛠️ Requirements

- **Podman** (for `podman pull`)
- **podlet** (for transpiling `compose` yaml into quadlet files)
- **systemd** with quadlet support
- Go 1.23+ (to build from source)

## 🚀 Installation

Build from source:
```bash
go build -o comquad ./cmd/comquad/
sudo cp comquad /usr/local/bin/

```

Or install directly via Go:

```bash
go install comquad/cmd/comquad@latest

```

## ⚙️ Usage

### Deploy a project

From a directory containing `compose` yaml:

```bash
comquad up

```

* **Override project name:** `comquad up -n my-service`
* **Force rebuild all images:** `comquad up --build`
* **Control image pulls:** `comquad up --pull [always|missing|never]` *(default: missing)*
* **Verbose output:** `comquad up -v` shows all transformations (port offsets, label additions, path normalizations, etc.)
* **Rootless port offset:** Set `ROOTLESS_PORT_OFFSET` env variable (default 2000) to shift privileged ports (< 1024) for rootless mode
* **Follow logs after deploy:** `comquad up -f` streams journal logs from the deployment timestamp until interrupted
* **Dry run:** `comquad up --dry-run` previews what would be deployed without writing any files or starting units
* **Quiet mode:** `comquad -q <command>` suppresses all non-error output (useful in scripts)

### Verbose Output

Use `-v` / `--verbose` with `up` to see every transformation comquad applies during deployment. Output uses ANSI colors (green=success, cyan=info, yellow=warning, blue=action) and respects `NO_COLOR`.

```bash
comquad up -v
```

Typical verbose output:

```
comquad: Injected container_name: myapp-web
comquad: Normalized image: nginx → docker.io/library/nginx
comquad: Normalized volume path: ./data → /home/user/project/data
comquad: Created default network: cq-default
comquad: Renamed web.container → cq-myapp-web.container
comquad: Rewrote cross-unit references in cq-myapp-web.container
comquad: Added AutoUpdate=registry to cq-myapp-web.container
comquad: Added [Install] section to cq-myapp-web.container
comquad: Added labels: Label=com.comquad.project=myapp, Label=com.comquad.managed=true
comquad: Offset port: PublishPort=80:80 → PublishPort=2080:80
comquad: Built image: myapp-web:latest
comquad: Successfully deployed project: myapp
```

### Dry Run

Use `--dry-run` to preview exactly what `comquad up` would generate and deploy, without touching the systemd directory or starting anything:

```bash
comquad up --dry-run
```

Typical output:

```
Dry run — project: myapp
Target directory: /home/user/.config/containers/systemd

[image] web          myapp-web:latest  (would build from /home/user/project)
[image] db           docker.io/library/postgres  (already exists locally, would skip pull)

2 quadlet file(s) would be written:

────────────────────────────────────────────────────────────
  /home/user/.config/containers/systemd/cq-myapp-web.container
────────────────────────────────────────────────────────────
[Container]
Image=myapp-web:latest
Network=cq-myapp-default.network
...

Dry run complete — nothing was written, no units started.
```

### Manage & Monitor

* **Start a project:** `comquad start [service ...]`
* **Stop a project:** `comquad stop [service ...]`
* **Restart a project:** `comquad restart [service ...]`
* **Remove a project:** `comquad down`
* **Remove with volumes:** `comquad down -d` (also removes Podman volumes)
* **List deployed projects:** `comquad list`
* **Regenerate state from labels:** `comquad regenerate --force`
* **Show unit status:** `comquad ps`
* **View unit file:** `comquad view [project] [service]`
* **Edit unit file:** `comquad edit [project] [service] [--no-reload]`
* **Check prerequisites:** `comquad check`

### Self-Healing

* **Regenerate state from Podman labels:** `comquad regenerate --force` (scans containers, networks, and volumes for `com.comquad.managed` label and reconstructs the state file)
* **Preview without writing:** `comquad regenerate --force --dry-run` (shows what would be regenerated without modifying the state file)

### Stream logs

```bash
comquad logs                  # all services (one-shot)
comquad logs -f               # all services (follow)
comquad logs web              # single service
comquad logs -n myapp web db  # specific project/services

```

For running units, logs are filtered to the current invocation. For stopped or failed units, full historical logs are shown.

### View

```bash
comquad view                  # view all units for a project
comquad view myapp            # view all units for a specific project
comquad view myapp web        # cat the cq-myapp-web.container file
comquad view -n myapp web     # override project name
```

### Edit

```bash
comquad edit                  # edit all units for a project
comquad edit myapp            # edit all units for a specific project
comquad edit myapp web        # edit the cq-myapp-web.container file
comquad edit -n myapp web     # override project name
comquad edit --no-reload      # open files without auto-reloading systemd
```

### Start, Stop & Restart

```bash
comquad start                 # start all units for a project
comquad start web             # start a specific service
comquad stop                  # stop all units for a project
comquad stop web db           # stop specific services
comquad restart               # restart all units for a project
comquad restart web           # restart a specific service
comquad -n myapp start        # override project name
```

### Exec

```bash
comquad exec web ls /app              # run command in web service
comquad exec web sh                   # interactive shell (TTY allocated by default)
comquad exec -u root web bash         # run as root inside the container
comquad exec web -- cat /etc/hostname # pass through to podman
```

## 🏗️ System Architecture & Internal Mechanics

For a deep dive into how `comquad` processes your compose files, manages state, and maps directories, please see dedicated [Architecture Guide](./ARCHITECTURE.md) guide.

## 📄 License

MIT
