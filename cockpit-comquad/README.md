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

## Testing

The plugin has a two-tier testing architecture:

### Unit Tests (vitest + React Testing Library)

Run the test suite:

```bash
npm test
```

This runs 60 tests across 7 test files covering:
- Client API functions
- All React components (App, ProjectsList, ProjectDetail, StackDeploy, DirectoryPicker, ErrorBoundary)
- Error handling and edge cases

Tests use mocked cockpit API for isolated testing without requiring a real cockpit environment.

### Browser Integration Tests (Selenium + Chromium)

Run browser-based integration tests:

```bash
# Build test container (requires comquad-test:latest)
make test-browser-image

# Run browser tests
make test-browser
```

Or from the repository root:

```bash
make integration-cockpit
```

The browser tests run in a containerized environment with:
- systemd as PID 1
- podman + comquad installed
- cockpit-ws running
- chromium-headless for automation

Tests cover:
- Stack dashboard page loads and displays projects
- Project detail view with services/containers/resources tabs
- Navigation and action buttons
- Status badges and refresh functionality

See [TESTING.md](./TESTING.md) for detailed testing documentation.

## Internationalization (i18n)

The plugin supports multiple languages using cockpit's gettext system. Translations are automatically loaded based on the browser's language setting.

### Supported Languages

The plugin includes translations for 20 core cockpit languages:
- Arabic (ar), Czech (cs), German (de), Spanish (es), Finnish (fi)
- French (fr), Indonesian (id), Italian (it), Japanese (ja), Georgian (ka)
- Korean (ko), Lao (lo), Polish (pl), Brazilian Portuguese (pt_BR), Romanian (ro)
- Russian (ru), Swedish (sv), Turkish (tr), Ukrainian (uk), Chinese Simplified (zh_CN)

### Adding New Translations

1. Copy the translation template:
   ```bash
   cp po/comquad.pot po/<lang>.po
   ```
   (e.g., `cp po/comquad.pot po/es.po` for Spanish)

2. Edit the `.po` file and translate the `msgstr` values

3. Add the language code to `po/LINGUAS`:
   ```bash
   echo "<lang>" >> po/LINGUAS
   ```

4. Build the translations:
   ```bash
   make po-build
   ```

This compiles `.po` files to JavaScript in `dist/po.<lang>.js`. The plugin automatically loads the appropriate translation file based on the browser's language.

### Extracting New Strings

If you add new translatable strings to the code:

1. Wrap strings with `_()`:
   ```typescript
   import { _ } from './i18n';
   const message = _('Hello, world!');
   ```

2. Update the translation template:
   ```bash
   make po-pot
   ```

3. Update existing translations:
   ```bash
   make po-update
   ```

### Translation Workflow

- `make po-pot` - Extract strings to `po/comquad.pot`
- `make po-update` - Update `.po` files from `.pot`
- `make po-build` - Compile `.po` to JavaScript for distribution

The build process generates:
- `po.<lang>.js` files containing `cockpit.locale()` calls with translation data
- `po.js` loader that auto-detects browser language and loads the appropriate translation

See [TESTING.md](./TESTING.md) for more details on the i18n implementation.

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
