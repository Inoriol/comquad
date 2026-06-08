# comquad

Docker-compose-like CLI for Podman Quadlets, backed by systemd.

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
3. **Cook** — prefixes files with `comquad-<project>`, rewrites cross-unit
   references, and applies rootless port offsets where needed
4. **Deploy** — copies files to the systemd config directory, registers project
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

## Project structure

```
cmd/comquad/        CLI entry point (cobra commands)
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

- `log` command — stream container logs via `podman logs`
- `edit` command — open the compose file in an editor and re-deploy
- Integration tests — end-to-end tests covering the full up/down lifecycle
- Smarter state management — reconcile `projects.json` with actual systemd units on disk

## License

MIT
