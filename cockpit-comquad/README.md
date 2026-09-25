# cockpit-comquad

Cockpit UI plugin for managing comquad stacks.

## Installation

### Prerequisites

- Cockpit installed and running
- comquad CLI installed (see [comquad installation](../README.md#installation))

### From Release Tarball

Download the latest release from [GitHub Releases](https://github.com/Inoriol/comquad/releases):

```bash
# Download and extract
wget https://github.com/Inoriol/comquad/releases/download/vX.Y.Z/cockpit-comquad-X.Y.Z.tar.gz
tar xf cockpit-comquad-X.Y.Z.tar.gz
cd cockpit-comquad-X.Y.Z

# Install for current user (no sudo required)
make user-install

# Or install system-wide (requires sudo)
sudo make install
```

### Uninstall

```bash
# User installation
make user-uninstall

# System-wide installation
sudo rm -rf /usr/local/share/cockpit/comquad
```

### Verify Installation

After installation, open Cockpit in your browser and look for "Comquad Stacks" in the sidebar.

## Overview

cockpit-comquad provides a web-based interface for managing Docker Compose projects deployed with comquad. It complements cockpit-podman by focusing on project/stack management rather than individual container operations.

## Features

- **Stack Dashboard**: View all deployed comquad projects with status, service counts, and source paths
- **Project Detail**: Inspect services, containers, and resources for each project
- **Stack Deployment**: Browse directories and deploy new compose.yaml projects
- **Deploy Preview**: Preview changes before deploying with expandable file diffs
- **Update Diff**: See what will change before applying updates
- **Lifecycle Management**: Start, stop, restart, and remove stacks with confirmation dialogs
- **Resource Viewer**: Click resource names to view generated quadlet file content
- **Clickable Ports**: Port mappings are clickable links with per-port http/https toggle
- **Resource Overview**: View networks, volumes, and images managed by each project

## Requirements

- Cockpit installed and running
- comquad CLI installed
- Node.js 18+ and npm for development

## Development

### Setup

```bash
cd cockpit-comquad
npm install
```

### Build

```bash
npm run build
```

### Development mode

```bash
npm run watch
```

### Install for development

```bash
make devel-install
```

This creates a symlink from `~/.local/share/cockpit/comquad` to the `dist/` directory. After making changes and rebuilding, refresh the Cockpit page in your browser.

### Uninstall development version

```bash
make devel-uninstall
```

## Architecture

The plugin communicates with comquad via `cockpit.spawn()` calls to the CLI with `--json` flag. It does not require a separate backend daemon.

### Components

- **App**: Main application with routing between views
- **ProjectsList**: Dashboard showing all deployed projects
- **ProjectDetail**: Detailed view of a single project with services, containers, resources, update diff preview, and resource unit viewer
- **StackDeploy**: Directory browser and deployment interface with change preview
- **client**: Wrapper around `cockpit.spawn()` calls to comquad CLI

### JSON API

The plugin relies on comquad's `--json` output mode for all commands:

- `comquad list --json` - List all projects
- `comquad ps --json` - List containers for a project
- `comquad view --json` - View project details
- `comquad view --json <resource>` - View individual resource unit file
- `comquad start/stop/restart --json` - Lifecycle management
- `comquad up --json --no-diff` - Deploy a stack
- `comquad up --json --dry-run` - Preview changes before deploying
- `comquad down --json -y` - Remove a stack
- `comquad logs --json` - View logs (batch mode)

## Design Decisions

- **No duplication with cockpit-podman**: This plugin focuses on project management. For container details, terminal access, and stats, users should navigate to cockpit-podman.
- **Manual refresh**: The UI does not auto-refresh. Users click the refresh button to update the view.
- **Stateless**: The UI always reads fresh state from the comquad CLI.
- **Directory browser**: The deploy interface uses a directory browser (not file picker) since compose projects are directory-based.

## License

MIT
