# comquad

ComQuad (Compose + Quadlet and small pun to kumquat) is Docker-compose-like CLI for Podman Quadlets, backed by systemd.

## 🚧 Project Status: Infra-Built Utility

I am an infrastructure engineer, not a full-time software developer. I built **Comquad** to solve a specific problem for my own workflow.

- **Contributions:** I am currently not accepting complex feature pull requests because I do not have the bandwidth or Go expertise to maintain them.
- **Bugs:** Feel free to open issues if a specific Docker Compose file breaks, but fixes will happen on a "best effort" timeline.
- **Philosophy:** This tool is intentionally small, simple, and transparent. It is not trying to become Kubernetes.

## Overview

`comquad` lets you define your services in a `compose.yaml` file and deploy them
as individual systemd units using Podman's Quadlet technology. Instead of running
its own orchestrator, comquad prepares the quadlet files and delegates lifecycle
management to systemd.

### How it works

1. **Preprocess** — normalizes your `compose.yaml` (absolute paths, default
   networks, project labels)
2. **Transpile** — runs `podlet` to convert the compose YAML into `.container`,
   `.network`, and `.volume` quadlet files
3. **Cook** — prefixes files with `cq-<project>`, rewrites cross-unit
   references, and applies rootless port offsets where needed
4. **Build** — builds images from `build:` contexts (if defined), checks local
   images, and pulls from registry (respects `--pull` flag)
5. **Deploy** — copies files to the systemd config directory, registers project
   state, and starts each unit via D-Bus

## Requirements

- **Podman** (for `podman pull`)
- **podlet** (for transpiling `compose.yaml` into quadlet files)
- **systemd** with quadlet support
- Go 1.23+ (to build from source)

## Installation

```bash
go build -o comquad ./cmd/comquad/
sudo cp comquad /usr/local/bin/
```

Or install directly:

```bash
go install comquad/cmd/comquad@latest
```

## Usage

### Deploy a project

From a directory containing `compose.yaml`:

```bash
comquad up
```

Override the project name:

```bash
comquad up -n my-service
```

Force rebuild all images:

```bash
comquad up --build
```

Control image pull behavior:

```bash
comquad up --pull always      # Always pull from registry
comquad up --pull missing      # Pull only if not found locally (default)
comquad up --pull never        # Fail if image not found locally
```

### Remove a project

```bash
comquad down
```

### List deployed projects

```bash
comquad list
```

### Check prerequisites

```bash
comquad check
```

### Stream logs

```bash
comquad logs                      # all services (one-shot)
comquad logs -f                   # all services (follow)
comquad logs web                  # single service
comquad logs -n myapp             # named project
comquad logs -n myapp web db      # specific services
```

### Show unit status

```bash
comquad ps                        # units for current project
comquad ps -n myapp               # units for named project
```

## Project structure

```
cmd/comquad/        CLI entry point (cobra commands)
internal/build/     Image building and pulling (podman build/pull)
internal/cooker/    Post-processes quadlet files (renaming, reference rewriting)
internal/deploy/    Systemd interaction (D-Bus), state management, target dir resolution
internal/orchestrator/ Wires all packages together, drives up/down lifecycle
internal/preprocess/  Normalizes compose.yaml (paths, networks, labels)
internal/transpile/   Runs the podlet binary to generate quadlet files
```

## Configuration

### State file

comquad stores project state in:

```
$XDG_DATA_HOME/comquad/projects.json
```

Default (if `XDG_DATA_HOME` is unset):

```
~/.local/share/comquad/projects.json
```

### Systemd target directory

- **Rootless** (non-root user): `~/.config/containers/systemd`
- **Root** (UID 0): `/etc/containers/systemd`

## compose.yaml format

comquad accepts standard Docker Compose v3 files. The following fields are
supported:

- `services` — container definitions
- `networks` — network definitions
- `volumes` — volume definitions

### Automatic behavior

- Relative volume paths are resolved to absolute paths
- A default bridge network is created if none are defined
- Services without explicit networks are attached to the default network
- Container names are auto-generated as `<project>-<service>` if not specified
- A `com.comquad.project` label is injected into every service
- Images without a registry prefix default to Docker Hub (`docker.io/library/`)
- Services with `build:` skip image normalization (no `docker.io/library/` prefix added)
- Build services are tagged as `<project>-<service>:latest`

### Build support

Services with a `build:` field are built locally using `podman build`. The image
is tagged as `<project>-<service>:latest` and used in the generated quadlet
files. Build configuration supports:

```yaml
services:
  web:
    build:
      context: ./apps/web        # Build context directory (default: .)
      dockerfile: Dockerfile.prod # Custom Dockerfile name (default: Dockerfile)
      target: production          # Build target stage
      args:                       # Build arguments
        VERSION: "1.0"
        ARCH: "amd64"
```

Or a shorthand string form:

```yaml
services:
  web:
    build: ./apps/web
```

Build images are not pulled from a registry. Use `--build` to force rebuild even
if the image exists locally.

### Opting out of AutoUpdate

Add `AutoUpdate=` (empty) to a service's quadlet output by setting the
`comquad-no-autoupdate` label:

```yaml
services:
  web:
    image: nginx
    labels:
      comquad-no-autoupdate: "true"
```

## TODO

- `ps` command — to get a clean status table
- `edit` command — open the quadlet file in an editor and re-deploy
- Integration tests — end-to-end tests covering the full up/down lifecycle
- Smarter state management — reconcile `projects.json` with actual systemd units on disk

## License

MIT
