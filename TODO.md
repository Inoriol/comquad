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

### Quadlet Directives in Compose Files

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

### Remaining Features

#### Logs viewer
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

#### YAML/Compose editor
- [ ] Add in-browser compose.yaml editor with syntax highlighting (Monaco or CodeMirror)
- [ ] Compose spec validation before deployment (use compose-go parser via new JSON API)
- [ ] Diff view showing changes before applying (reuse existing dry-run preview)
- [ ] `.env` file editor alongside compose.yaml
- [ ] Save + deploy workflow (edit → validate → preview → deploy)
- [ ] Syntax error highlighting and inline validation messages
- [ ] Auto-save drafts to prevent data loss
- [ ] Add `comquad validate --json` command to validate compose files without deploying
- [ ] Unit tests for editor component and validation flow

#### Resource metrics
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

#### Shell/Exec integration
- [ ] Add "Open Terminal" button to service rows in ProjectDetail.tsx
- [ ] Deep-link to cockpit-podman container detail page with terminal
- [ ] Construct cockpit-podman URL: `/podman/#/containers/<container-id>/console`
- [ ] Handle cases where cockpit-podman is not installed (show message)
- [ ] Add "Exec Command" quick action for one-off commands (optional enhancement)
- [ ] Unit tests for URL generation and link handling

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

---

## 🔄 Podman Compose Compatibility Layer & Provider

### Implementation Plan

#### Phase 1: Orchestrator Changes ✅

**1.1 Compose file path support** ✅
- Add `composePath string` field to `Orchestrator`
- Add `WithComposeFile(path string)` method
- If path is file → use directly; if directory → search (current behavior)

**1.2 Refactor to options structs** ✅
```go
type UpOptions struct {
    PullStrategy    string
    Follow          bool
    DryRun          bool
    NoDiff          bool
    ForceRebuild    bool
    SkipBuild       bool
    ForceRecreate   bool
    SkipRecreate    bool
    RemoveOrphans   bool
    Services        []string  // For service filtering
}

type DownOptions struct {
    RemoveVolumes   bool
    DryRun          bool
    RemoveOrphans   bool
}

type PullOptions struct {
    PullStrategy    string  // default: "always"
    DryRun          bool
    Services        []string
}
```

**1.3 Implement orphan detection** ⏳ (Deferred)
- Query podman for containers with project label
- Compare with services in current compose file
- Stop containers not in compose file
- Integrate into `Up()` and `Down()`

**1.4 Implement service filtering** ⏳ (Deferred)
- Filter units at start time based on `Services` list
- Include dependencies if needed

**1.5 Add `Pull()` method** ✅
- Transpile, reconcile, write files
- Start `.image` units only (filtered by services if specified)
- Skip builds and containers

**1.6 Update existing methods** ✅
- `Up(opts UpOptions) error`
- `Build(opts UpOptions) error` (reuse UpOptions minus some fields)
- `Down(opts DownOptions) error`

#### Phase 2: New Commands ✅

**2.1 `comquad pull` command** ✅
- Pull images defined in compose.yaml
- Support service filtering: `pull [service...]`
- Default pull strategy: `always`

**2.2 `comquad compose` parent command** ✅
- Registers all compose subcommands
- Handles global flags (`--file`, `-p`, `--dry-run`)

**2.3 Compose subcommands (P0)** ✅

| Command | Translation |
|---------|-------------|
| `compose up` | `Up{Follow: true}` |
| `compose up -d` | `Up{Follow: false}` |
| `compose up --build` | `Up{ForceRebuild: true}` |
| `compose up --no-build` | `Up{SkipBuild: true}` |
| `compose up --force-recreate` | `Up{ForceRecreate: true}` |
| `compose up --no-recreate` | `Up{SkipRecreate: true}` |
| `compose up --remove-orphans` | `Up{RemoveOrphans: true}` |
| `compose down` | `Down{}` |
| `compose down --volumes` | `Down{RemoveVolumes: true}` |
| `compose down --remove-orphans` | `Down{RemoveOrphans: true}` |
| `compose ps` | `Ps()` passthrough |
| `compose logs` | `Logs()` passthrough |

**2.4 Compose subcommands (P1)** ✅

| Command | Translation |
|---------|-------------|
| `compose start/stop/restart` | Passthrough |
| `compose exec` | Invert `-T` → `tty=false` |
| `compose build` | Passthrough |
| `compose pull` | Passthrough to `pull` command |

#### Phase 3: Flag Translation Layer ✅

Create `cmd/comquad/compose_flags.go` with translation functions.

#### Phase 4: Provider Mode ✅

- Detect invocation as `docker-compose` or `podman-compose` ✅
- Skip `compose` subcommand requirement ✅
- Reuse translation layer ✅

#### File Structure ✅

```
cmd/comquad/
├── compose.go           # Parent "compose" command
├── compose_flags.go     # Flag translation functions
├── compose_up.go        # compose up subcommand
├── compose_down.go      # compose down subcommand
├── compose_ps.go        # compose ps subcommand
├── compose_logs.go      # compose logs subcommand
├── compose_start.go     # compose start subcommand
├── compose_stop.go      # compose stop subcommand
├── compose_restart.go   # compose restart subcommand
├── compose_exec.go      # compose exec subcommand
├── compose_build.go     # compose build subcommand
├── compose_pull.go      # compose pull subcommand
├── provider.go          # Provider mode detection
└── pull.go              # comquad pull command

internal/orchestrator/
├── orchestrator.go      # Update Up(), Build() signatures
├── down.go              # Update Down() signature
├── pull.go              # New Pull() method
└── options.go           # New UpOptions, DownOptions, PullOptions structs
```

---

### Concept

Make comquad work as a **drop-in replacement** for docker-compose/podman-compose, allowing users to use familiar docker-compose CLI syntax while getting comquad's unique systemd-native workflow (diffs, confirmations, quadlet generation).

### Research: How `podman compose` Works

`podman compose` is a **thin wrapper** that delegates to external compose providers:

1. **Provider Discovery** (in order):
   - `PODMAN_COMPOSE_PROVIDER` environment variable
   - `containers.conf` → `[engine].compose_providers` setting
   - Default: `docker-compose` (preferred) or `podman-compose`

2. **Environment Setup**:
   - Sets `DOCKER_HOST` to Podman's socket (e.g., `unix:///run/user/1000/podman/podman.sock`)
   - Sets `DOCKER_BUILDKIT=0` (Podman doesn't support all buildkit features)
   - Passes through `DOCKER_CONFIG`

3. **Execution**: Passes all args directly to the provider, preserving exit codes

**Key insight**: `podman compose` doesn't implement compose itself - it just sets up the environment and calls an external tool (docker-compose or podman-compose).

### Two-Phase Approach

#### Phase 1: Compatibility Layer (`comquad compose`)
```bash
comquad compose up -d          # Familiar docker-compose syntax
comquad compose logs -f        # But uses comquad's systemd workflow internally
comquad compose down -v        # Same flags, comquad behavior
```

**Goal**: Users type `comquad compose` instead of `docker compose`
- Accepts docker-compose CLI syntax
- Translates to comquad commands internally
- Keeps comquad's unique features (diffs, confirmations, systemd-native)

#### Phase 2: Provider Mode (`podman compose`)
```bash
PODMAN_COMPOSE_PROVIDER=comquad podman compose up
# podman compose calls comquad as if it were docker-compose
```

**Goal**: comquad becomes a provider that `podman compose` can call
- Uses the same translation logic from Phase 1
- Exposed as a provider binary that podman can invoke
- Users can use `podman compose` as usual, but comquad handles it

### Why This Order?

1. **Lower risk** - Test the translation logic in isolation first
2. **Prove the concept** - Show docker-compose syntax works with comquad's workflow
3. **Iterate safely** - Fix edge cases before exposing as a provider
4. **User choice** - Some may prefer `comquad compose`, others `podman compose`

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│              docker-compose CLI flag parser                 │
│              (translates DC flags → comquad commands)       │
└─────────────────────────────────────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
    ┌─────────────────┐      ┌─────────────────────┐
    │ comquad compose │      │ podman compose      │
    │ up -d           │      │ (provider mode)     │
    └─────────────────┘      └─────────────────────┘
              │                         │
              └────────────┬────────────┘
                           ▼
              ┌─────────────────────────┐
              │  Existing comquad       │
              │  orchestrator.Up()      │
              │  (systemd/quadlet)      │
              └─────────────────────────┘
```

### Commands to Support (Priority Order)

| docker-compose Command | comquad Equivalent | Priority | Notes |
|------------------------|-------------------|----------|-------|
| `up` | `comquad up` | P0 | Map `-d`→default, no flag→`-f` |
| `down` | `comquad down` | P0 | Map `-v`→`-d` |
| `ps` | `comquad ps` | P0 | Map `-a` |
| `logs` | `comquad logs` | P0 | Map `-f`, `--tail`, `--since`, `-t` |
| `start` | `comquad start` | P1 | |
| `stop` | `comquad stop` | P1 | |
| `restart` | `comquad restart` | P1 | |
| `exec` | `comquad exec` | P1 | Map `-u`, `-T` |
| `build` | `comquad build` | P1 | |
| `pull` | (new) | P2 | Pull images only |
| `config`/`convert` | (new) | P2 | Validate & print compose |
| `images` | (new) | P3 | List images |
| `kill` | (skip) | - | Use `stop` instead |
| `run` | (skip) | - | Use `exec` instead |
| Others | (defer) | - | Future if needed |

### Flag Mapping

#### Global Flags (before command)
| docker-compose | comquad | Notes |
|----------------|---------|-------|
| `-f, --file FILE` | (handle) | Set compose file path |
| `-p, --project-name NAME` | `-n NAME` | Project name |
| `--profile NAME` | (defer) | Not yet supported |
| `--env-file FILE` | (defer) | Not yet supported |
| `--dry-run` | `--dry-run` | Same |
| `--progress TYPE` | (ignore) | comquad has own output |

#### `up` Command Flags
| docker-compose | comquad | Notes |
|----------------|---------|-------|
| `-d, --detach` | (default) | comquad returns to shell by default |
| (no flag) | `-f` | Attach = follow logs in comquad |
| `--build` | (handle) | Force rebuild |
| `--no-build` | (handle) | Skip build |
| `--pull always/missing/never` | `--pull STRATEGY` | Same |
| `--force-recreate` | (handle) | Always recreate |
| `--no-recreate` | (handle) | Never recreate |
| `--remove-orphans` | (handle) | Remove unmanaged |
| `--timeout SECONDS` | (defer) | Timeout for stop |
| `-t, --timestamps` | `-t` in logs | Show timestamps |

#### `down` Command Flags
| docker-compose | comquad | Notes |
|----------------|---------|-------|
| `-v, --volumes` | `-d` | Remove volumes |
| `--rmi all/local` | (defer) | Remove images |
| `--remove-orphans` | (handle) | Remove unmanaged |
| `-t, --timeout SECONDS` | (defer) | Timeout |

#### `logs` Command Flags
| docker-compose | comquad | Notes |
|----------------|---------|-------|
| `-f, --follow` | `-f` | Same |
| `--tail N` | `--tail N` | Same |
| `--since TIME` | `--since TIME` | Same |
| `-t, --timestamps` | `-t` | Same |
| `--no-color` | (handle) | Respect NO_COLOR |

### Implementation Plan

#### Phase 1: Compatibility Layer

**New Files:**
```
cmd/comquad/
├── compose.go              # Parent "compose" command
├── compose_up.go           # compose up subcommand
├── compose_down.go         # compose down subcommand  
├── compose_ps.go           # compose ps subcommand
├── compose_logs.go         # compose logs subcommand
├── compose_start.go        # compose start subcommand
├── compose_stop.go         # compose stop subcommand
├── compose_restart.go      # compose restart subcommand
├── compose_exec.go         # compose exec subcommand
├── compose_build.go        # compose build subcommand
└── compose_pull.go         # compose pull subcommand (new functionality)
```

**Key Implementation Details:**

1. **compose.go** - Parent command that:
   - Registers all `compose *` subcommands
   - Handles global flags (`-f`, `-p`, `--dry-run`)
   - Sets up compose file path and project name context

2. **Flag Translation Layer:**
   ```go
   // Example: compose up -d --pull always
   // 1. Parse: detach=true, pull="always"
   // 2. Translate: follow=false, pullStrategy="always"
   // 3. Call: orchestrator.Up("always", false, dryRun, false)
   ```

3. **Compose File Handling:**
   - Support `-f` flag for alternate compose files
   - Support `COMPOSE_FILE` env var
   - Default lookup: `compose.yaml`, `docker-compose.yaml`, `podman-compose.yaml`

4. **Output Compatibility:**
   - Show similar progress indicators
   - Container names in logs (prefixed with service name)
   - Exit codes match docker-compose behavior

#### Phase 2: Provider Mode

**Implementation:**
- Add `--provider-mode` flag or detect when invoked as `docker-compose`/`podman-compose`
- When in provider mode:
  - Accept docker-compose args directly (no `compose` subcommand)
  - Set up DOCKER_HOST environment if needed
  - Use same translation logic from Phase 1

**Provider mode:**
```bash
# Set as podman compose provider:
export PODMAN_COMPOSE_PROVIDER=comquad
podman compose up
```

### Behavioral Differences (Documented)

| Aspect | docker-compose | comquad compose |
|--------|----------------|-----------------|
| Default `up` | Start & attach | Start & return (like `-d`) |
| `up` no flag | Attach to logs | Same as `-d` |
| `up` to attach | Use no flag | Use `-f` |
| Changes | Apply immediately | Show diff & confirm |
| Skip confirm | N/A | Use `--no-diff` |
| Service manager | Direct container ops | systemd (via quadlet) |
| Container names | `project-service-1` | `project-service` (systemd) |
| Restart behavior | Docker restart | systemd restart |

### Testing Strategy

1. **Unit Tests:**
   - Flag parsing and translation logic
   - Compose file path resolution
   - Project name handling

2. **Integration Tests:**
   - `comquad compose up` creates same resources as `comquad up`
   - `comquad compose down` removes same resources
   - Flag combinations work correctly
   - Multiple compose files (`-f file1 -f file2`)

3. **Compatibility Tests:**
   - Run existing docker-compose projects with `comquad compose`
   - Verify expected behavior
   - Test with real-world compose files

4. **Provider Mode Tests:**
   - Set `PODMAN_COMPOSE_PROVIDER=comquad`
   - Run `podman compose` commands
   - Verify comquad handles them correctly

### Future Extensions

- `--profile` support (when comquad supports profiles)
- `--env-file` support
- `docker-compose.yml` version field handling
- `run` command (one-off commands)
- `watch` command (file watching)
- `scale` command (when comquad supports scaling)

### References

- podman compose source: https://github.com/containers/podman/blob/main/cmd/podman/compose.go
- docker compose docs: https://docs.docker.com/compose/reference/
- podman-compose: https://github.com/containers/podman-compose
