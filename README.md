# comquad

`comquad` (Compose + Quadlet) is a Docker Compose-like CLI for Podman Quadlets and systemd.

It reads a standard `compose.yaml`, generates Quadlet unit files, and lets systemd manage the resulting services. It is intended to provide a Compose-style workflow while using Podman and systemd underneath.

![comquad demo](.github/assets/demo.gif)

## Quick Start

Install `comquad`, then run it from a directory containing `compose.yaml` (also supports `docker-compose.yaml` and `podman-compose.yaml`):

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

After changing `compose.yaml`, run `comquad up` again. `comquad` compares the new generated units with the deployed project, displays a diff, and asks for confirmation before applying changes. It restarts only services affected by the changes.

Use `comquad up --no-diff` to apply changes without displaying the diff or asking for confirmation.

## Requirements

- Podman 4.4 or newer (4.8+ recommended for native `.image` quadlet support)
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

`comquad up` flow is inspired by Terraform. The Compose file is the input, Quadlet files are the generated configuration, and each `up` compares the desired configuration with what is already deployed before applying changes.

`comquad` does not run its own service supervisor. It generates Quadlets and delegates service lifecycle management to systemd.

## Commands

Run commands from the directory containing the project `compose.yaml`.

| Command | Purpose | Common options |
|---|---|---|
| `comquad up` | Generate or update Quadlets and start the project | `-f` follow logs, `--dry-run` preview, `--pull always\|missing\|never`, `--no-diff` skip diff and confirmation |
| `comquad build` | Build images defined in compose.yaml without starting containers | `-f` follow build logs, `--dry-run` preview, `--pull always\|missing\|never`, `--no-diff` skip diff and confirmation |
| `comquad pull [service ...]` | Pull images defined in compose.yaml without starting containers | `--dry-run` preview |
| `comquad down` | Stop and remove the project | `-d` also remove named volumes, `-y` skip confirmation, `--dry-run` preview |
| `comquad ps` | Show container status | `-a` include exited containers |
| `comquad start [service ...]` | Start all services or selected services | `--dry-run` preview |
| `comquad stop [service ...]` | Stop all services or selected services | `--dry-run` preview |
| `comquad restart [service ...]` | Restart all services or selected services | `--dry-run` preview |
| `comquad logs [service]` | Show service logs | `-f` follow, `--tail N`, `--since TIME`, `-t` show timestamps |
| `comquad exec SERVICE COMMAND` | Run a command in a running container | `-u USER`, `-t` control TTY |
| `comquad compose COMMAND` | Docker Compose compatible commands | See [Compose Compatibility](#compose-compatibility) |
| `comquad view [RESOURCE]` | Show the project or a generated Quadlet file | `comquad view web` |
| `comquad edit [SERVICE]` | Edit generated Quadlet files in `$EDITOR` | `--no-reload` do not reload systemd |
| `comquad list [PROJECT]` | List deployed projects, or view a specific project | Also available as `comquad ls` |
| `comquad regenerate --force` | Rebuild local project state from managed Podman resources | `--dry-run` preview |
| `comquad check` | Check required tools and system configuration | |

Every command has more details and examples in its help output:

```bash
comquad up --help
comquad logs --help
comquad exec --help
```

## Compose Compatibility

comquad provides a Docker Compose compatible command layer via `comquad compose`. This allows users familiar with docker-compose to use the same syntax while getting comquad's unique features like diffs, confirmations, and systemd-native workflow.

```bash
comquad compose up -d          # Deploy in background
comquad compose logs -f        # Follow logs
comquad compose down --volumes # Remove with volumes
comquad compose exec web bash  # Execute in container
```

### Supported Commands

| Command | Description |
|---------|-------------|
| `compose up` | Create and start containers |
| `compose down` | Stop and remove containers, networks |
| `compose ps` | List containers |
| `compose logs` | View output from containers |
| `compose start` | Start services |
| `compose stop` | Stop services |
| `compose restart` | Restart services |
| `compose exec` | Execute a command in a running container |
| `compose build` | Build or rebuild services |
| `compose pull` | Pull service images |

### Key Differences from Docker Compose

| Aspect | docker-compose | comquad compose |
|--------|----------------|-----------------|
| Default `up` | Start & attach | Start & return (like `-d`) |
| `up` to attach | Use no flag | Use `-f` |
| `down -v` | Remove volumes | Use `--volumes` |
| Changes | Apply immediately | Show diff & confirm |
| Skip confirm | N/A | Use `--no-diff` |
| Service manager | Direct container ops | systemd (via quadlet) |

## Compose Files

comquad accepts standard Compose files with `services`, `networks`, and `volumes`. Compose services, networks, volumes, secrets, images, and build blocks are translated into the corresponding Quadlet units where supported.

Some behavior is handled automatically:

- Relative bind-mount paths are converted to absolute paths.
- A default network is added when a service has no explicit network.
- Service and project labels are added to managed resources.
- Rootless privileged ports are shifted by `ROOTLESS_PORT_OFFSET`.
- Image and build units are generated for systemd to manage.
- SELinux labels added to volumes: `:Z` (private) for single-service volumes, `:z` (shared) for volumes used by multiple services.
- Secrets are managed via Podman's native secret store. Environment secrets are written to temporary files at deployment time and loaded via `podman secret create`. File secrets are loaded directly. Secrets are mounted at `/run/secrets/<name>` in containers. Secrets support custom `target`, `uid`, `gid`, and `mode` options (Podman 4.5.0+). Secrets can be exposed as environment variables using `x-type: env` extension.

Native Quadlet directives that have no Compose equivalent can be set through x-extensions. Supported extensions: `x-container`, `x-image`, `x-build`, `x-network`, `x-volume`, and `x-systemd` (for `[Service]` and `[Unit]` sections). X-extensions override compose-generated directives when there's a conflict.

```yaml
services:
  web:
    image: nginx
    x-container:
      Timezone: "Europe/Berlin"
    x-image:
      AllTags: true
    x-build:
      DNS: "8.8.8.8"
    x-systemd:
      Service:
        MemoryMax: "512M"

networks:
  frontend:
    x-network:
      ContainersConfModule: "/etc/containers/my.conf"

volumes:
  data:
    x-volume:
      User: "1000"
```

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the complete Compose-to-Quadlet mapping and implementation details.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `EDITOR` | Auto-detected | Editor used by `comquad edit`. |
| `NO_COLOR` | Unset | Set to any value to disable ANSI colors. |
| `ROOTLESS_PORT_OFFSET` | `2000` | Offset applied to privileged ports in rootless mode. |
| `XDG_DATA_HOME` | `~/.local/share` | Base directory for `comquad` state and deployment data. |
| `COMQUAD_SELINUX_DANGEROUS_PATHS` | `/etc,/var,/home,/usr,/sys,/proc,/boot,/root` | Comma-separated list of host paths that trigger a warning when bind-mounted with SELinux relabeling. |
| `COMQUAD_SKIP_REGISTRY_NORMALIZATION` | Unset | Set to any value to skip image name normalization (respects `registries.conf`). |
| `COMQUAD_SYSTEMD_SPECIFIERS` | Unset | Set to any value to use systemd specifiers (e.g., `%h`) for paths under `$HOME`. |
| `COMQUAD_AUTONS` | Unset | Set to `1` to enable AutoNS mode for user namespace isolation (rootful only). |

## AutoNS Mode (Rootful)

comquad supports optional user namespace isolation between stacks using Podman's AutoNS feature. When enabled, each stack gets a dedicated 65535-wide UID/GID range, ensuring containers within a stack share one user namespace while containers across stacks are fully isolated.

### Enabling AutoNS

```bash
export COMQUAD_AUTONS=1
sudo comquad up
```

AutoNS is **rootful-only** and requires Podman 4.4.0+ for `idmap` volume support.

### System Configuration

Add a `containers` entry to `/etc/subuid` and `/etc/subgid`:

```
containers:10000000:6553600
```

This provides space for ~100 stacks with 65535 UIDs each.

### How It Works

1. **Anchor selection**: One container per stack is designated as the "anchor" (root of `depends_on` DAG, or alphabetically first service)
2. **UID allocation**: comquad allocates a 65535-wide UID range from `/etc/subuid` for the stack
3. **Namespace sharing**: The anchor gets `--userns=auto:uidmapping=0:<base>:65535`, other containers join via `--userns=container:<anchor>`
4. **Volume idmap**: All volumes get the `idmap` option for proper UID mapping

### Isolation Guarantees

- Containers **within** a stack share one user namespace → shared volumes work naturally
- Containers **across** stacks are in different UID ranges → fully isolated from each other
- Host filesystem access is mapped through the UID range, providing an additional security layer

See [TODO.md](./TODO.md) for implementation details and design decisions.

## Files and State

Generated Quadlet files are stored in:

- Rootless mode: `~/.config/containers/systemd`
- Root mode: `/etc/containers/systemd`

`comquad` stores project state below `$XDG_DATA_HOME/comquad/`. Successful deployments also keep a baseline used to show diffs and preserve manual changes made with `comquad edit`.

Use `comquad regenerate --force` if the local state file is lost but the managed Podman resources and Quadlet files still exist.

## Project Status

`comquad` is an infrastructure utility built for a specific workflow. Issues and Compose compatibility reports are welcome, but fixes are best effort.

## Cockpit Plugin

A web-based UI plugin for Cockpit is available in the `cockpit-comquad/` directory. It provides:

- Stack dashboard with project status and service counts
- Project detail view with services, containers, and resources
- Stack deployment via directory browser with change preview
- Lifecycle management (start/stop/restart/remove) with update diff preview
- Resource unit file viewer (click resource names to view quadlet content)
- Clickable port links with per-port http/https toggle
- Internationalization (i18n) support with gettext
- Comprehensive test suite (unit tests + browser integration tests)

See [cockpit-comquad/README.md](./cockpit-comquad/README.md) for details.

## License

MIT
