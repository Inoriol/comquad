# comquad

`comquad` is a Docker Compose-like CLI for Podman Quadlets and systemd.

It reads a standard `compose.yaml`, generates Quadlet unit files, and lets systemd manage the resulting services. It is intended to provide a Compose-style workflow while using Podman and systemd underneath.

![comquad demo](.github/assets/demo.gif)

## Quick Start

Install comquad, then run it from a directory containing `compose.yaml`:

```bash
comquad up
```

This is similar to `docker compose up -d`: it deploys the project and returns to the shell.

To keep the terminal attached and follow service logs, use:

```bash
comquad up -f
```

This is similar to `docker compose up` without `-d`.

Stop and remove the project with:

```bash
comquad down
```

After changing `compose.yaml`, run `comquad up` again. comquad compares the new generated units with the deployed project, displays a diff, and asks for confirmation before applying changes. It restarts only services affected by the changes.

Use `comquad up --no-diff` to apply changes without displaying the diff or asking for confirmation.

## Requirements

- Podman 4.8 or newer
- systemd with Quadlet support
- Go 1.25 or newer when building from source

## Installation

Build from source:

```bash
go build -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o comquad ./cmd/comquad/
sudo cp comquad /usr/local/bin/
```

Or install directly with Go:

```bash
go install github.com/Inoriol/comquad/cmd/comquad@latest
```

Check the installation:

```bash
comquad --version
```

## Design

comquad is inspired by Terraform. The Compose file is the input, Quadlet files are the generated configuration, and each `up` compares the desired configuration with what is already deployed before applying changes.

comquad does not run its own service supervisor. It generates Quadlets and delegates service lifecycle management to systemd.

## Commands

Run commands from the directory containing the project `compose.yaml`.

| Command | Purpose | Common options |
|---|---|---|
| `comquad up` | Generate or update Quadlets and start the project | `-f` follow logs, `--dry-run` preview, `--pull always\|missing\|never`, `--no-diff` skip diff and confirmation |
| `comquad down` | Stop and remove the project | `-d` also remove named volumes, `-y` skip confirmation, `--dry-run` preview |
| `comquad ps` | Show container status | `-a` include exited containers |
| `comquad start [service ...]` | Start all services or selected services | `--dry-run` preview |
| `comquad stop [service ...]` | Stop all services or selected services | `--dry-run` preview |
| `comquad restart [service ...]` | Restart all services or selected services | `--dry-run` preview |
| `comquad logs [service]` | Show service logs | `-f` follow, `--tail N`, `--since TIME`, `-t` show timestamps |
| `comquad exec SERVICE COMMAND` | Run a command in a running container | `-u USER`, `-t` control TTY |
| `comquad view [RESOURCE]` | Show the project or a generated Quadlet file | `comquad view web` |
| `comquad edit [SERVICE]` | Edit generated Quadlet files in `$EDITOR` | `--no-reload` do not reload systemd |
| `comquad list` | List deployed projects | Also available as `comquad ls` |
| `comquad regenerate --force` | Rebuild local project state from managed Podman resources | `--dry-run` preview |
| `comquad check` | Check required tools and system configuration | |

Every command has more details and examples in its help output:

```bash
comquad up --help
comquad logs --help
comquad exec --help
```

## Compose Files

comquad accepts standard Compose files with `services`, `networks`, and `volumes`. Compose services, networks, volumes, secrets, images, and build blocks are translated into the corresponding Quadlet units where supported.

Some behavior is handled automatically:

- Relative bind-mount paths are converted to absolute paths.
- A default network is added when a service has no explicit network.
- Service and project labels are added to managed resources.
- Rootless privileged ports are shifted by `ROOTLESS_PORT_OFFSET`.
- Image and build units are generated for systemd to manage.

Native Quadlet behavior can also be added through supported `comquad-*` labels. For example:

```yaml
services:
  web:
    image: nginx
    labels:
      comquad-no-autoupdate: "true"
```

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the complete Compose-to-Quadlet mapping and implementation details.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `EDITOR` | Auto-detected | Editor used by `comquad edit`. |
| `NO_COLOR` | Unset | Set to any value to disable ANSI colors. |
| `ROOTLESS_PORT_OFFSET` | `2000` | Offset applied to privileged ports in rootless mode. |
| `XDG_DATA_HOME` | `~/.local/share` | Base directory for comquad state and deployment data. |

## Files and State

Generated Quadlet files are stored in:

- Rootless mode: `~/.config/containers/systemd`
- Root mode: `/etc/containers/systemd`

comquad stores project state below `$XDG_DATA_HOME/comquad/`. Successful deployments also keep a baseline used to show diffs and preserve manual changes made with `comquad edit`.

Use `comquad regenerate --force` if the local state file is lost but the managed Podman resources and Quadlet files still exist.

## Project Status

comquad is an infrastructure utility built for a specific workflow. Issues and Compose compatibility reports are welcome, but fixes are best effort.

## License

MIT
