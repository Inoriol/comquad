## 🗺️ Roadmap & Next Steps

For long-term goals, refer to the [roadmap](./ROADMAP.md).

### Follow-ups

- **Value-level merge** — Multi-value directives (`Environment=`, `Volume=`, `PublishPort=`) are merged at key granularity; independent additions to the same key currently conflict instead of being merged per line.
- **Diff rendering** — The unified diff implementation is homegrown; it could be replaced with `go-udiff` if richer output is ever needed.

### Possible Security Improvements

- **Tmpfs-backed secrets via `LoadCredential`** — Currently, secrets are bind-mounted directly from managed files on disk. Consider generating a companion `.service` unit (rather than a `.container` Quadlet) that uses systemd `LoadCredential=` in `[Service]`, combined with `Volume=%d/<name>`, to mount secrets from systemd's RAM-backed credential directories (`/run/credentials/`). This would keep secret values in tmpfs memory rather than on persistent storage. An initial implementation using Quadlet's `[Service]` pass-through did not integrate `LoadCredential=` and `Volume=` with credential paths correctly in the container lifecycle. A standalone `.service` file could bypass Quadlet entirely for credential setup.

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
- [ ] Create `src/LogsView.tsx` component (deferred - lower priority)
- [ ] Display log entries in monospace font
- [ ] Filter by service (dropdown)
- [ ] Filter by priority (info/warning/error)
- [ ] Auto-scroll toggle
- [ ] Timestamp display
- [ ] Fetch via `client.projectLogs()` or direct `journalctl` call

#### 2.8 Styling
- [x] Import PatternFly styles in index.tsx
- [x] Status badge colors (healthy=green, stopped=gray, degraded=orange, failed=red)

#### 2.9 Testing
- [ ] Set up test infrastructure (test VM with comquad installed)
- [ ] Integration tests using Chrome DevTools Protocol
- [ ] Test scenarios: list projects, view detail, start/stop, deploy
- [ ] Follow cockpit-podman test patterns

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
13. ⏳ Phase 2.7: Logs viewer (deferred - lower priority)
14. ⏳ Phase 2.9: Testing (ongoing, but formalize at end)

### References

- cockpit-podman: https://github.com/cockpit-project/cockpit-podman
- Cockpit starter-kit: https://github.com/cockpit-project/starter-kit
- PatternFly: https://www.patternfly.org/
- Cockpit documentation: https://cockpit-project.org/guide/latest/

