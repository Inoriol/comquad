## 🗺️ Roadmap & Next Steps

For long-term goals, refer to the [roadmap](./ROADMAP.md).

### Follow-ups

- **Value-level merge** — Multi-value directives (`Environment=`, `Volume=`, `PublishPort=`) are merged at key granularity; independent additions to the same key currently conflict instead of being merged per line.
- **Diff rendering** — The unified diff implementation is homegrown; it could be replaced with `go-udiff` if richer output is ever needed.

### Packaging & Distribution

- **Homebrew tap (Linux-only)** — Create separate `homebrew-comquad` repo with two formulas:
  - `comquad` formula: CLI-only, downloads Linux binaries from main repo releases, `depends_on :linux` to restrict to Linux. Architectures: amd64, arm64 (no risc-v — homebrew doesn't properly support it).
  - `cockpit-comquad` formula: Cockpit plugin that uses Homebrew's `post_install_steps` with `sudo: true` to create a symlink from `(share/"cockpit-comquad")` to `/usr/share/cockpit/comquad`. Depends on `comquad` formula (auto-installs CLI first). Uses modern Homebrew feature designed for system-path installations.
  - Automate formula updates via GitHub Actions on main repo release.

- **deb packages** — Use goreleaser `nfpms` to build two packages:
  - `comquad` package: Go binary + man pages, architectures: amd64, arm64, riscv64
  - `cockpit-comquad` package: cockpit plugin web assets installed to `/usr/share/cockpit/comquad/`, `Depends: comquad`, `Recommends: cockpit` (soft dep — accommodates containerized cockpit)
  - Architectures: amd64, arm64, riscv64

- **rpm packages** — Use goreleaser `nfpms` to build two packages:
  - `comquad` package: Go binary + man pages, architectures: x86_64, aarch64, riscv64
  - `cockpit-comquad` package: cockpit plugin web assets, `Requires: comquad`, `Suggests: cockpit` (soft dep)
  - Architectures: x86_64, aarch64, riscv64

- **RISC-V architecture** — Add riscv64 to goreleaser build targets for binaries and deb/rpm packages only (skip for brew)

### Tmpfs Secrets (`.cq-secret`) — ✅ Implemented

- **Goal:** Implement secure secret handling via `.cq-secret` definitions that are regular `.service` files with a custom extension, compiled into companion systemd `.service` units using `LoadCredential=`.

- **Mechanism:**
  - Parse `.cq-secret` definitions from compose files
  - Generate `.cq-secret` files (regular `.service` files with custom extension) that stage secrets in RAM-backed tmpfs (`/run/credentials/`)
  - Quadlet generator copies `.cq-secret` files to `.service` files in the systemd directory
  - Pass secrets into containers via dynamic `podman secret` injection or volume mounts
  - Keep sensitive data off persistent disk entirely

- **Upstream potential:** Acts as production PoC for proposing native `.secret` Quadlet type to `containers/podman`

- **Complexity:** High (~500-800 lines, systemd credential integration, security review required)

- **Status:** ✅ Implemented. Default behavior uses tmpfs staging. Secrets with custom `uid`/`gid`/`mode` fall back to volume mounts.

### Pseudo-Quadlet Fallback (Lower Baseline to 4.4.0) — ✅ Implemented

- **Goal:** Lower minimum Podman baseline from 4.8.0 → **4.4.0** by generating `.cq-image` and `.cq-build` files (regular `.service` files with custom extensions) when native quadlet types are unavailable, which are then copied to `.service` files by the quadlet generator.

- **Scope:**
  - `.cq-image` (native since 4.8.0) → fallback to `.cq-image` (`.service` file) with `ExecStart=podman image pull ...` for Podman 4.4.0-4.7.x
  - `.cq-build` (native since 5.2.0) → fallback to `.cq-build` (`.service` file) with `ExecStart=podman build ...` for Podman 4.4.0-5.1.x

- **Implementation location:** `compose2quadlet` library (mapper-level decision)

- **Behavior:**
  - Automatic based on `WithPodmanVersion()` configuration
  - Option to disable fallback (e.g., `WithoutPseudoQuadlet()`)
  - Emit warning when fallback is active
  - Quadlet generator copies `.cq-*` files to `.service` files in the systemd directory

- **Fallback service characteristics:**
  - `Type=oneshot` with `RemainAfterExit=yes`
  - Container units reference via `After=<name>.service` instead of `Image=<name>.cq-image`
  - For `.cq-build` fallback: `WorkingDirectory=` set to build context
  - For `.cq-image` fallback: `--pull=always/missing/never` logic in `ExecStart` command

- **Status:** ✅ Implemented. Pseudo-quadlet generator converts `.cq-image` and `.cq-build` to `.service` units with appropriate `ExecStart=` commands.

- **Out of scope:**
  - Podman < 4.4.0 support (no quadlet at all)
  - Runtime version detection (already handled)

- **Documentation updates:**
  - ✅ Update `compose2quadlet/doc/mapping.md` baseline from 4.8.0 → 4.4.0
  - ✅ Add section documenting pseudo-quadlet fallback behavior
  - ✅ Fill in "?" version columns in mapping.md

### Quadlet Directives in Compose Files

- **X-extension** — ✅ Implemented. Supports `x-container`, `x-image`, `x-build`, `x-network`, `x-volume`, and `x-systemd` (for `[Service]` and `[Unit]` sections). X-extensions override compose-generated directives. Validates directive names and warns on unknown directives.

- **Timers** — Service-level timer support via `x-timer`. Design decisions:
  - **Target**: `Target: "build"|"service"` (defaults to `"build"` if service has `build:`, else `"service"`)
  - **Restart behavior**: `RestartService: true` adds `BindsTo=` to container unit, causing automatic restart when build completes
  - **Warnings**:
    - Build timer without `RestartService: true` warns that container won't use rebuilt image
    - `Target: "service"` with `build:` present warns about stale image
  - **Structure**:
    ```yaml
    services:
      web:
        build: .
        x-timer:
          OnCalendar: "daily"
          Persistent: true
          Target: "build"
          RestartService: true
    ```
  - **Generated units**: `<service>.timer` with `[Timer]` section, targeting `.build` or `.service` unit
  - **Restart mechanism**: Uses `BindsTo=` (native systemd dependency) rather than `ExecStartPost=`
  - **Multiple timers**: Not supported (keep it simple)

- **Questionable future feature: Round-trip** — Make it possible to convert a project back into a Compose file.

---

## 🎛️ Cockpit Plugin (cockpit-comquad)

Web UI plugin for Cockpit to manage comquad projects as stacks, inspired by Portainer's stack management.

### Design Decisions

- **Backend**: Pure `cockpit.spawn()` calling `comquad --json`. No daemon.
- **State**: Stateless UI — always reads fresh from CLI.
- **Auth**: Handled by Cockpit.
- **Rootless/root**: Follow cockpit-podman pattern (detect user context).
- **Error handling**: All errors as JSON error envelopes.
- **Real-time**: Manual refresh first, auto-refresh later.
- **Web console**: Skip — link to cockpit-podman for container details/terminal.
- **Folder picker**: Custom directory browser widget (not file picker).
- **Location**: `cockpit-comquad/` subdirectory in this repo.

### MVP Features

- List all projects (stacks) with status, service counts, source paths
- Project detail view with services, resources, quadlet files
- View container configuration (generated quadlet files)
- Logs viewer with filtering by service/priority
- Deploy new stack via folder browser
- Start/stop/restart/remove stacks
- Link to cockpit-podman for container details/terminal/stats

### Phase 1: JSON API Additions (Prerequisites)

#### 1.1 Add `--json` to `view` command
- [x] Add `ViewData`, `ServiceJSON`, `ResourceJSON` schemas to `internal/output/schemas.go`
- [x] Refactor `viewProject()` in `internal/orchestrator/view.go` to build structured data
- [x] Add `output.IsJSONMode()` branch to `viewProject()`
- [x] Add `output.IsJSONMode()` branch to `viewUnit()` for single file viewing
- [x] Test with `comquad view --json`

#### 1.2 Add `--json` to `start/stop/restart` commands
- [x] Add `LifecycleData` schema to `internal/output/schemas.go`
- [x] Add JSON branches to `Start()`, `Stop()`, `Restart()` in `internal/orchestrator/lifecycle.go`
- [x] Test with `comquad start --json`, `comquad stop --json`, `comquad restart --json`

#### 1.3 Add `--json` to `down` command
- [x] Add `DownData` schema to `internal/output/schemas.go`
- [x] Add JSON branch to `Down()` in `internal/orchestrator/down.go`
- [x] Test with `comquad down --json`

#### 1.4 Add `--json` to `up` command
- [x] Add `UpData` schema to `internal/output/schemas.go`
- [x] Add JSON branch to `Up()` in `internal/orchestrator/orchestrator.go`
- [x] Suppress logger output in JSON mode for pipeline functions
- [x] Test with `comquad up --json --no-diff`

#### 1.5 Add `--json` to `logs` command
- [x] Add `LogsData`, `LogEntryJSON` schemas to `internal/output/schemas.go`
- [x] Add JSON branch to `Logs()` in `internal/orchestrator/log.go` (batch mode only)
- [x] Test with `comquad logs --json`

#### 1.6 Error handling in JSON mode
- [x] Wrap command execution in `cmd/comquad/main.go` to catch errors
- [x] Output JSON error envelope when `output.IsJSONMode()` and command fails
- [x] Test error cases with `--json` flag

### Phase 2: cockpit-comquad Plugin

#### 2.1 Project scaffold
- [x] Create `cockpit-comquad/` directory
- [x] Create `package.json` with React, PatternFly, TypeScript dependencies
- [x] Create `tsconfig.json` for TypeScript configuration
- [x] Create `build.js` (copy from cockpit-podman, adapt for comquad)
- [x] Create `Makefile` with build targets (build, install, devel-install, watch)
- [x] Create `.eslintrc.json` and `.stylelintrc.json`
- [x] Create `src/manifest.json` with plugin registration
- [x] Create `src/index.html` entry point

#### 2.2 TypeScript types
- [x] Create `src/types.ts` mirroring Go JSON schemas
- [x] Define interfaces: `Project`, `Container`, `ViewData`, `Service`, `Resource`, `LogEntry`, etc.
- [x] Define API response types: `ListResponse`, `PSResponse`, `LifecycleResponse`, etc.

#### 2.3 Client wrapper
- [x] Create `src/client.ts` with functions wrapping `cockpit.spawn()`
- [x] Implement `listProjects()`, `projectPs()`, `viewProject()`
- [x] Implement `startProject()`, `stopProject()`, `restartProject()`, `removeProject()`
- [x] Implement `deployProject()`, `projectLogs()`
- [x] Handle JSON parsing and error extraction

#### 2.4 Projects list dashboard
- [x] Create `src/ProjectsList.tsx` component
- [x] PatternFly `Table` or `Gallery` showing projects
- [x] Display: name, status badge, services count, source path
- [x] Add manual refresh button
- [x] Click project → navigate to detail view

#### 2.5 Project detail view
- [x] Create `src/ProjectDetail.tsx` component
- [x] Header section: project name, status, source path, action buttons
- [x] Services table: name, status, image, networks, volumes
- [x] Resources section: networks, volumes, images lists
- [x] Containers tab: container list with state, image, ports
- [x] Action buttons with confirmation modals: start, stop, restart, down

#### 2.6 Stack deploy dialog
- [x] Create `src/StackDeploy.tsx` component
- [x] Directory browser: breadcrumb navigation, folder list, compose.yaml detection
- [x] Path input with navigation
- [x] Deploy button → calls `client.deployProject()`
- [x] Show deployment success/error

#### 2.7 Logs viewer
- [ ] Create `src/LogsView.tsx` component
- [ ] Real-time log streaming via `cockpit.spawn()` with streaming output
- [ ] Display log entries in monospace font with syntax highlighting
- [ ] Filter by service (dropdown)
- [ ] Filter by priority (info/warning/error)
- [ ] Search within logs (text input with highlight)
- [ ] Auto-scroll toggle (follow new logs)
- [ ] Timestamp display with timezone handling
- [ ] Fetch via `client.projectLogs()` or direct `journalctl` call
- [ ] Handle log rotation and reconnection
- [ ] Add unit tests for log streaming and filtering

#### 2.8 Styling
- [x] Import PatternFly styles in index.tsx
- [x] Status badge colors (healthy=green, stopped=gray, degraded=orange, failed=red)

#### 2.9 Testing
- [x] Set up test infrastructure (vitest + React Testing Library for unit tests)
- [x] Unit tests: 60 tests across 7 test files (client, components)
- [x] Browser integration tests (Selenium + Chromium in container)
- [x] Follow cockpit-podman test patterns (adapted for comquad)

#### 2.10 JSON API: `up --dry-run --json`
- [x] Add `DryRunData`, `DryRunImage`, `DryRunBuild`, `DryRunFile` schemas to `internal/output/schemas.go`
- [x] Add JSON branch to `printDryRun()` in `internal/orchestrator/images.go`
- [x] Return structured data: project, target_dir, pull_strategy, images, builds, files (with diff), has_changes
- [x] Test with `comquad up --dry-run --json`

#### 2.11 Cockpit plugin enhancements
- [x] Add dry-run preview to StackDeploy.tsx (Preview Changes button before deploy)
- [x] Add diff preview to Update confirmation modal in ProjectDetail.tsx
- [x] Add resource unit viewer modal (click resource name to view quadlet content)
- [x] Add clickable port links with per-port http/https toggle (lock/unlock icon)
- [x] Fix Makefile manifest.json path (src/manifest.json)

#### 2.12 YAML/Compose editor
- [ ] Add in-browser compose.yaml editor with syntax highlighting (Monaco or CodeMirror)
- [ ] Compose spec validation before deployment (use compose-go parser via new JSON API)
- [ ] Diff view showing changes before applying (reuse existing dry-run preview)
- [ ] `.env` file editor alongside compose.yaml
- [ ] Save + deploy workflow (edit → validate → preview → deploy)
- [ ] Syntax error highlighting and inline validation messages
- [ ] Auto-save drafts to prevent data loss
- [ ] Add `comquad validate --json` command to validate compose files without deploying
- [ ] Unit tests for editor component and validation flow

#### 2.13 Resource metrics
- [ ] **Prerequisite: New JSON API** - Add `comquad stats --json` using Podman stats
  - [ ] Add `StatsData`, `ServiceStats` schemas to `internal/output/schemas.go`
  - [ ] Add JSON branch to orchestrator stats method
  - [ ] Return structured data: CPU %, memory usage/limit, network I/O, block I/O
  - [ ] Support both single project and all projects mode
  - [ ] Test with `comquad stats --json`
- [ ] Create `src/ResourceMetrics.tsx` component
- [ ] CPU/memory usage graphs per service in project detail view (PatternFly charts)
- [ ] Real-time updates via polling or streaming
- [ ] Historical data collection (optional - start with point-in-time metrics)
- [ ] Resource usage summary cards (total CPU, memory, disk)
- [ ] Alert indicators for high resource usage (>80% threshold)
- [ ] Unit tests for metrics component and data formatting

#### 2.14 Shell/Exec integration
- [ ] Add "Open Terminal" button to service rows in ProjectDetail.tsx
- [ ] Deep-link to cockpit-podman container detail page with terminal
- [ ] Construct cockpit-podman URL: `/podman/#/containers/<container-id>/console`
- [ ] Handle cases where cockpit-podman is not installed (show message)
- [ ] Add "Exec Command" quick action for one-off commands (optional enhancement)
- [ ] Unit tests for URL generation and link handling

### Execution Order

1. ✅ Phase 1.1: `view --json` (validates schema design)
2. ✅ Phase 1.2: `start/stop/restart --json` (validates lifecycle pattern)
3. ✅ Phase 1.3: `down --json` (simple)
4. ✅ Phase 1.6: Error handling in JSON mode (critical for plugin UX)
5. ✅ Phase 2.1-2.3: Plugin scaffold + types + client (foundation)
6. ✅ Phase 2.4: Projects list (first visible feature, validates end-to-end)
7. ✅ Phase 2.5: Project detail (core feature)
8. ✅ Phase 1.4: `up --json` (needed for deploy)
9. ✅ Phase 2.6: Stack deploy (depends on `up --json`)
10. ✅ Phase 1.5: `logs --json` (needed for logs viewer)
11. ✅ Phase 2.10: `up --dry-run --json` (needed for preview features)
12. ✅ Phase 2.11: Cockpit plugin enhancements (preview, resource viewer, clickable ports)
13. ✅ Phase 2.9: Testing (unit tests + browser integration tests)
14. ✅ Internationalization (i18n) - all strings wrapped, po/ infrastructure, build integration
15. ⏳ Phase 2.7: Logs viewer (real-time streaming, search, filtering)
16. ⏳ Phase 2.12: YAML/Compose editor (in-browser editing with validation)
17. ⏳ Phase 2.13: Resource metrics (needs `stats --json` API first)
18. ⏳ Phase 2.14: Shell/Exec integration (link to cockpit-podman terminal)

### References

- cockpit-podman: https://github.com/cockpit-project/cockpit-podman
- Cockpit starter-kit: https://github.com/cockpit-project/starter-kit
- PatternFly: https://www.patternfly.org/
- Cockpit documentation: https://cockpit-project.org/guide/latest/

---

## 🍺 Appstore (brew-like app registry)

Brew-like appstore where anyone can host a git repo with apps that users can add to their comquad installation.

### Design Decisions

- **CLI-first**: Design CLI interface first, GUI will follow
- **Hierarchical structure**: Each app in its own directory with compose.yaml + optional metadata
- **Copy on install**: Apps copied to local project directory (consistent with existing workflow)
- **Per-app tracking**: Each installed app has its own app.json for origin/version tracking
- **Type validation**: CLI-side validation for environment variables (cockpit uses --json output)
- **Metadata optional**: Apps work without metadata; metadata enables better GUI experience

### Commands

#### Repository Management
```bash
comquad repo add <url> [--name <alias>] [--branch <branch>]
comquad repo remove <name|url>
comquad repo list
comquad repo update [name]
```

#### App Management
```bash
comquad app search <query>
comquad app install <app> [--version <version>] [--env KEY=VALUE ...] [-y]
comquad app info <app>
comquad app upgrade <app>
comquad app list [--installed|--available]
comquad app remove <app>
```

### Directory Structure

#### Repository Structure (source)
```
my-comquad-apps/
├── repo.yaml          # Optional repo metadata
├── apps/
│   ├── nginx-simple/
│   │   ├── compose.yaml
│   │   └── app.yaml   # Optional app metadata
│   ├── wordpress/
│   │   ├── compose.yaml
│   │   ├── app.yaml
│   │   └── .env.example
```

#### Installed Apps (local)
```
~/.local/share/comquad/apps/
├── wordpress/
│   ├── app.json      # { "repo": "...", "app": "wordpress", "version": "v1.0.0", "installed_at": "..." }
│   ├── compose.yaml
│   └── .env
└── nginx-simple/
    ├── app.json
    └── compose.yaml
```

### Metadata Format

#### repo.yaml (optional, repo-level)
```yaml
name: "Community Apps"
description: "Curated collection of comquad apps"
maintainer: "John Doe"
homepage: "https://github.com/..."
```

#### app.yaml (optional, app-level, GUI-focused)
```yaml
name: "WordPress"
description: "WordPress CMS with MySQL"
icon: "icon.svg"
categories: ["cms", "blog"]

# GUI field definitions + env defaults
environment:
  MYSQL_ROOT_PASSWORD:
    description: "MySQL root password"
    type: "password"
    required: true
  SITE_TITLE:
    description: "Site title"
    default: "My WordPress"
  PORT:
    description: "HTTP port"
    type: "number"
    default: 8080
```

### Environment Variable Handling

**Interactive mode (default):**
- Prompts for required variables
- Shows defaults for optional variables
- User can accept defaults or provide values

**Non-interactive mode:**
- `-y` flag: skip prompts, use defaults (fail if required missing)
- `--env KEY=VALUE`: provide specific values
- Can combine: `comquad app install wordpress -y --env MYSQL_ROOT_PASSWORD=secret`

**Type validation (CLI-side):**
- `password`: masked input, no validation
- `number`: numeric validation
- `url`: URL format validation
- Default (no type): string, no validation

### Versioning

- **Git tags** as source of truth (e.g., `wordpress-v1.0.0`)
- **Version field** in app.yaml optional (for display when no tag)
- Tag takes precedence if both exist

### Installation Flow

1. `comquad app install wordpress`
2. If `app.yaml` has environment vars → CLI prompts for required ones (or uses defaults with `-y`)
3. Copy `compose.yaml` to `~/.local/share/comquad/apps/wordpress/`
4. Write provided env vars to `.env` in project dir
5. Write `app.json` with repo origin and version
6. Run `comquad up` in that directory

### Future Scope (deferred)

- **x-init extension**: Post-install setup commands (init containers)
- **App dependencies**: Declare dependencies between apps
- **App groups**: Install multiple related apps together
- **Custom cockpit UI**: Render custom fields in cockpit plugin based on app.yaml

