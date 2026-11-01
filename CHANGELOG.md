# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2026-11-01

### Added

- **Pseudo-quadlet service naming convention** — Pseudo-quadlet services now follow a consistent naming convention that matches native quadlet services:
  - `.cq-image` → `-image.service` (matches native `.image` → `-image.service`)
  - `.cq-build` → `-build.service` (matches native `.build` → `-build.service`)
  - `.cq-secret` → `-secret.service` (comquad-specific, no native equivalent)
  - This ensures service names remain consistent when Podman is upgraded and native types become available
- **Container references for pseudo-quadlets** — Container units now use actual image references with `After=`/`Requires=` dependencies for pseudo-quadlets:
  - Native: `Image=web.image` (quadlet translates to dependency)
  - Pseudo: `Image=docker.io/library/nginx:latest` + `After=web-image.service` + `Requires=web-image.service`
  - Added `pseudoQuadletDeps()` in `mapper/unit.go` to emit `After=`/`Requires=` for pseudo-quadlet dependencies
  - Updated `ApplyReferences()` to convert `.cq-image` → `-image.service` and `.cq-build` → `-build.service` in dependency directives
- **Pseudo-quadlet fallback for `.image` and `.build` units** — Lowered minimum Podman baseline from 4.8.0 to 4.4.0 by generating `.cq-image` and `.cq-build` pseudo-units that are converted to `.service` files when native quadlet types are unavailable.
  - `.cq-image` fallback for Podman 4.4.0-4.7.x (native `.image` requires 4.8.0+)
  - `.cq-build` fallback for Podman 4.4.0-5.1.x (native `.build` requires 5.2.0+)
  - New `WithoutPseudoQuadlet()` option to disable fallback behavior
  - New unit types: `UnitCQImage`, `UnitCQBuild`
  - Pseudo-quadlet generator converts `.cq-*` units to `.service` units with appropriate `ExecStart=` commands
  - Container units reference pseudo-services via `After=` instead of `Image=`
- **Pseudo-Quadlet Generator** — New `compose2quadlet/pseudo/` package converts custom unit types (`.cq-*`) to `.service` units
- **Secure Secret Handling** — Secrets now staged in RAM-backed tmpfs via systemd `LoadCredential=` (never written to persistent disk)
  - Environment secrets: `LoadCredential=<name>:/bin/sh -c 'printf "%s" "$ENV_VAR"'`
  - File secrets: `LoadCredential=<name>:<path>`
  - Custom permissions (`uid`/`gid`/`mode`): Falls back to volume mount
- **Secret Deduplication** — Shared secrets across services produce only one `.cq-secret` unit
- **Integration Tests** — 6 new test cases covering environment secrets, file secrets, custom permissions, deduplication, multiple secrets, and custom target paths

### Changed

- **Edit command pseudo-quadlet handling** — `comquad edit` now properly handles pseudo-quadlet files:
  - After editing `.cq-*` files, copies content to corresponding `.service` file
  - Includes `.service` files in systemd daemon reload
  - Restarts pseudo-quadlet services when changed
  - Added `copyPseudoQuadlets()` and `removePseudoQuadletServices()` to `pipeline.go`
- **Logs command pseudo-quadlet support** — `comquad logs` and `comquad logs -f` now include pseudo-quadlet services:
  - Updated `FollowLogs()` to include `.cq-secret`, `.cq-image`, `.cq-build` files
  - Added `CQImageFileToUnitName()` and `CQBuildFileToUnitName()` helpers
- **View command pseudo-quadlet display** — `comquad view` now displays pseudo-quadlets as resources:
  - Shows `.cq-secret`, `.cq-image`, `.cq-build` in the RESOURCES table
  - Updated `shortName()` to strip pseudo-quadlet extensions
- **Lifecycle commands pseudo-quadlet support** — `start`, `stop`, `restart` now include pseudo-quadlet files:
  - Updated `resolveUnits()` to include `.cq-secret`, `.cq-image`, `.cq-build` when no services specified
  - Updated matching logic to handle pseudo-quadlet files by name
- **Handle images/builds for pseudo-quadlets** — Extended `handleImages()` and `handleBuilds()` to process `.cq-image` and `.cq-build` files:
  - Pseudo-image services always pull (no `Policy=` support)
  - Pseudo-build services always rebuild
- **JSON output for pseudo-quadlets** — Added `Secrets` field to `ResourcesJSON` and `ProjectJSON`:
  - `comquad list --json` now includes secrets in project resources
  - Updated `types.ts` in cockpit-comquad to include `secrets` field
- **Reconcile pseudo-quadlet support** — Updated `isQuadletFile()` to recognize `.cq-image` and `.cq-build` files
- **Resolve pseudo-quadlet support** — Updated `quadletResourceSuffixes` to include `.cq-secret`, `.cq-image`, `.cq-build`
- **Minimum Podman version**: 4.8.0 → 4.4.0 (with pseudo-quadlet fallback)
- Updated `compose2quadlet/doc/mapping.md` baseline documentation from 4.8.0 to 4.4.0
- Filled in ~40 uncertain version ("?") entries in `mapping.md` with verified versions
- Build units now generate `.cq-build` fallback instead of fatal error for Podman < 5.2.0
- Image units now generate `.cq-image` fallback instead of being skipped for Podman < 4.8.0
- **Dead code removal** — Removed unused `resolveImageRef()` and `hasBuildUnit()` functions from `images.go`
- **Refactored Secret Handling** — `PremapSecrets()` now returns `([]Directive, []QuadletUnit)` to support pseudo-quadlet generation
- **Reference Rewriting** — Added `.secretref` placeholder handling in `ApplyReferences()` for correct secret mount paths
- **State Tracking** — Added `Secrets []string` to `ResourceInfo` for tracking managed secrets
- Updated `README.md` requirements section to reflect Podman 4.4+ minimum
- Updated `ARCHITECTURE.md` with pseudo-quadlet generator details for `.cq-image` and `.cq-build`
- Updated `compose2quadlet/ARCHITECTURE.md` with expanded pseudo-quadlet generator documentation
- Updated `compose2quadlet/README.md` package description to mention new pseudo-quadlet types
- Added comprehensive pseudo-quadlet fallback section to `compose2quadlet/doc/mapping.md`
- Marked pseudo-quadlet fallback task as completed in `TODO.md`
- Updated README.md, ARCHITECTURE.md, TODO.md with secret handling details
- Updated compose2quadlet README.md and ARCHITECTURE.md with pseudo-quadlet generator docs
- Added comprehensive secret handling section to compose2quadlet/ARCHITECTURE.md

### Fixed

- **Pseudo-quadlet type preservation** — Fixed pseudo-conversion to preserve original type (`cq-secret`, `cq-image`, `cq-build`) instead of converting to "service". This allows state tracking and file management to work correctly.
- **Secret mount path** — Updated secret volume mount path to use `-secret.service` suffix: `/run/credentials/cq-project-db_password-secret.service/db_password`
- **Secret Mount Path** — Fixed incorrect `%i` specifier expansion in container mount directives
  - Before: `/run/credentials/%i/<name>` (incorrect)
  - After: `/run/credentials/<prefixed-service>.service/<name>` (correct)
- **Pseudo-quadlet lifecycle consistency** — Fixed several inconsistencies in how pseudo-quadlets (`.cq-image`, `.cq-build`) were handled compared to native quadlets:
  - `fileToUnitName()` now handles `.cq-image` and `.cq-build` cases (previously returned empty string, breaking lifecycle operations)
  - `Down()` now stops `.cq-image` and `.cq-build` services during teardown (previously left them running)
  - `hasBuildUnitForName()` and `hasImageUnitForName()` now check both native and pseudo types (previously caused incorrect behavior when determining if a service has a corresponding build/image unit)
  - `resolveUnits()` now guards against empty unit names as a defensive measure
- **Pseudo-quadlet naming consolidation** — `copyPseudoQuadlets()` and `removePseudoQuadletServices()` now use `CQ*FileToUnitName()` helper functions instead of duplicating inline name construction logic
- **Secret directive validation** — Added `knownSecretDirectives` map and `SectionSecret` case in `isValidDirective()` so x-extension validation recognizes valid secret directives (`Source=`, `Environment=`, `Path=`)

### Security

- Secrets no longer written to persistent disk (except custom-permissions fallback)
- `/run/credentials/` is tmpfs-backed and auto-cleaned on service stop
- Container mounts are read-only (`:ro`)

### Backward Compatibility

- External secrets continue to use native Podman `Secret=` directive
- Secrets with custom `uid`/`gid`/`mode` fall back to volume mount (preserves existing behavior)
- Existing compose files work without modification

### Testing

- Added unit tests for `.cq-image` conversion (`pseudo/image_test.go`)
- Added unit tests for `.cq-build` conversion (`pseudo/build_test.go`)
- Updated `TestBuilds_VersionGate` to test pseudo-quadlet fallback behavior
- Added `TestBuilds_VersionGate_NoPseudo` to test behavior when fallback is disabled
- Added `TestVersion_Build_PseudoQuadlet` to test build pseudo-quadlet fallback across versions
- Added `TestVersion_Image_PseudoQuadlet` to test image pseudo-quadlet fallback across versions
- Updated container mapper to respect `PseudoQuadlet` flag when generating image references
- Updated pseudo-conversion tests to expect preserved types
- Updated secret volume reference test for new `-secret.service` naming
- Updated integration test helpers for new secret service naming
- Added `TestFileToUnitName` covering all 8 file types (`.container`, `.network`, `.volume`, `.image`, `.build`, `.cq-secret`, `.cq-image`, `.cq-build`) plus unknown extensions
- Added `TestDown_StopsPseudoImageAndBuildUnits` verifying that `Down()` stops all pseudo-quadlet services
- Updated `TestHasBuildUnitForName` and `TestHasImageUnitForName` to verify pseudo-type matching (`UnitCQBuild`, `UnitCQImage`)

## [0.4.4] - 2026-09-27

### Added

- **Internationalization (i18n) support** — Full i18n support for the cockpit-comquad plugin using cockpit's gettext system. The plugin now supports 20 core cockpit languages with automatic language detection based on browser settings.
  - Created `cockpit-comquad/po/comquad.pot` translation template with 79 translatable strings
  - Added 20 language translation files (`.po`): Arabic (ar), Czech (cs), German (de), Spanish (es), Finnish (fi), French (fr), Indonesian (id), Italian (it), Japanese (ja), Georgian (ka), Korean (ko), Lao (lo), Polish (pl), Brazilian Portuguese (pt_BR), Romanian (ro), Russian (ru), Swedish (sv), Turkish (tr), Ukrainian (uk), Chinese Simplified (zh_CN)
  - Created `cockpit-comquad/po/LINGUAS` file listing all supported languages
  - Created `cockpit-comquad/po/POTFILES.in` listing source files with translatable strings
- **i18n helper functions** — Created `cockpit-comquad/src/i18n.ts` with translation helper functions:
  - `_()`: Wrapper for `cockpit.gettext()` with template string support
  - `ngettext()`: Wrapper for `cockpit.ngettext()` for plural forms
  - `format()`: Wrapper for `cockpit.format()` for string formatting
- Updated all source files to use `_()` for translatable strings:
  - `src/app.tsx`: 9 strings (UI titles, buttons, messages)
  - `src/ProjectsList.tsx`: 6 strings (table headers)
  - `src/ProjectDetail.tsx`: 40+ strings (tabs, buttons, modals, messages)
  - `src/StackDeploy.tsx`: 15+ strings (deployment UI, preview)
  - `src/DirectoryPicker.tsx`: 7 strings (directory browser)
  - `src/ErrorBoundary.tsx`: 3 strings (error messages)
- Updated `cockpit-comquad/src/index.html` to load `po.js` for translation initialization
- Updated `cockpit-comquad/src/client.ts` to use global `cockpit` object instead of imports
- **i18n build system** — Updated `cockpit-comquad/build.js` to compile `.po` files to JavaScript format:
  - Generates `po.<lang>.js` files with `cockpit.locale()` calls
  - Generates `po.js` loader for automatic language detection
  - Uses proper gettext format with plural-forms and language-direction metadata
- Updated `cockpit-comquad/Makefile` with i18n targets:
  - `po-pot`: Extract translatable strings to template
  - `po-update`: Update `.po` files from template
  - `po-build`: Compile `.po` to JavaScript for distribution
- Created `cockpit-comquad/po/compile.py` helper script for PO to JS compilation
- **i18n tests** — Created `cockpit-comquad/test/i18n.test.ts` with 8 tests for i18n module:
  - Tests for `_()` function with and without template variables
  - Tests for `ngettext()` function for plural forms
  - Tests for `format()` function with template variables
- All 68 unit tests pass (60 original + 8 new i18n tests)
- Updated README.md with i18n section: list of supported languages, instructions for adding new translations, extracting new strings, and translation workflow commands
- Updated ARCHITECTURE.md with i18n implementation details
- Updated TODO.md to mark i18n task as complete

### Changed

- Updated `cockpit-comquad/test/setup.ts` to mock cockpit i18n functions
- Test mock provides no-op translation functions for English testing

## [0.4.3] - 2026-09-25

### Added

- **JSON output for `comquad up --dry-run`** — New `DryRunData` schema with structured preview data including project name, target directory, pull strategy, images (with pull actions), builds (with image tags), and file changes (with unified diffs). Enables machine-readable dry-run output for automation and UI integration.
- New `DryRunData`, `DryRunImage`, `DryRunBuild`, `DryRunFile` types in `internal/output/schemas.go`.
- New `dryRunProject()` function in cockpit-comquad `client.ts`.
- New TypeScript interfaces `DryRunData`, `DryRunImage`, `DryRunBuild`, `DryRunFile` in cockpit-comquad `types.ts`.
- **Cockpit plugin: Deploy preview** — "Preview Changes" button in Stack Deploy dialog shows dry-run output before deploying. Displays images, builds, and expandable file diffs with status labels (created/changed/removed).
- **Cockpit plugin: Update diff preview** — Update confirmation modal now auto-fetches and displays dry-run diff, showing file changes before applying updates.
- **Cockpit plugin: Resource unit viewer** — Clicking resource names in the Resources tab opens a modal showing the full quadlet unit file content (networks, volumes, images, builds).
- **Cockpit plugin: Clickable port links** — Container ports in the Containers tab are now clickable links. Each port has a lock/unlock icon to toggle between http and https schemes per-port.
- **Cockpit plugin: Improved UX** — Loading spinners, expandable sections, and better error handling throughout.

### Changed

- **`printDryRun()` refactored** — Split into text and JSON branches. JSON mode returns early with structured data via `output.PrintJSON()`.
- **Cockpit plugin: StackDeploy.tsx** — Added preview state management, expandable file sections, and loading indicators.
- **Cockpit plugin: ProjectDetail.tsx** — Added dry-run fetching in update modal, resource viewer modal, and per-port scheme toggle with lock/unlock icons.
- **Makefile** — Fixed `manifest.json` path to `src/manifest.json` in `install` and `user-install` targets.

### Fixed

- `comquad up --dry-run --json` now produces JSON output instead of plain text. Previously, the `--json` flag was ignored during dry-run.
- Cockpit plugin Makefile now correctly copies `src/manifest.json` instead of looking for `manifest.json` in the root directory.

### Testing

- Existing unit tests pass with new JSON dry-run code path.
- Manual testing verified: `comquad up --dry-run --json` produces valid JSON with expected structure.

## [0.4.1] - 2026-09-24

### Added

- **Enhanced `comquad ls` command** — Now displays project status (healthy/degraded/stopped/up) and service count (running/total) using go-pretty table for better readability.
- **`comquad ls <project>` shortcut** — Passing a project name as positional argument now delegates to `comquad view -n <project>`, showing detailed project information.
- **Status and Services fields in JSON output** — `ProjectJSON` now includes `status` and `services` fields for machine-readable output.
- **Registry normalization control** — New `COMQUAD_SKIP_REGISTRY_NORMALIZATION` environment variable to skip image name normalization and respect `registries.conf`. When set, bare image names (e.g., `nginx`) are not automatically expanded to `docker.io/library/nginx`, allowing Podman's native registry resolution to take precedence.
- **Systemd specifier support** — New `COMQUAD_SYSTEMD_SPECIFIERS` environment variable to enable path shortening using systemd specifiers. When set, bind-mount paths under `$HOME` are shortened using `%h` specifier in generated quadlet files.
- **Differentiated SELinux labeling** — SELinux volume labeling now distinguishes between shared and non-shared volumes. Volumes used by multiple services receive `:z` (shared) label, while volumes used by a single service receive `:Z` (private) label. This provides better security isolation by default.
- New `WithoutImageNormalization()` option in compose2quadlet library to disable image name normalization.
- New `WithSystemdSpecifiers()` option in compose2quadlet library to enable systemd specifier path shortening.
- New `compose2quadlet/opinionated/specifiers.go` implementing `ApplySpecifiers()` transform for `%h` path shortening.
- Shared volume analysis in `compose2quadlet/transpile.go` via `analyzeSharedVolumes()` function to determine which volumes are referenced by multiple services.
- **JSON API expansion** — Added JSON output for various commands:
  - `comquad view --json`: Added JSON output for project overview (`ViewData`) and individual unit files (`UnitFileJSON`).
  - `comquad start/stop/restart --json`: Added `LifecycleData` schema for lifecycle action results.
  - `comquad down --json`: Added `DownData` schema with removed files, networks, and volumes.
  - `comquad up --json`: Added `UpData` schema with deployment summary (files written, units started).
  - `comquad logs --json`: Added `LogsData` schema with structured log entries (batch mode only).
  - Error handling: All commands now return JSON error envelopes when `--json` is used and an error occurs.
  - Suppressed operational logger output in JSON mode for all pipeline functions (prepareUnits, startContainers, handleImages, handleBuilds) to keep stdout clean for JSON parsing.
- **Cockpit plugin** — New `cockpit-comquad/` directory: Web-based UI plugin for Cockpit to manage comquad projects as stacks.
  - Stack dashboard: View all deployed projects with status, service counts, and source paths.
  - Project detail view: Inspect services, containers, and resources with three tabs (Services, Containers, Resources).
  - Stack deployment: Directory browser interface for deploying new compose.yaml projects.
  - Lifecycle management: Start, stop, restart, and remove stacks with confirmation dialogs.
  - React + PatternFly + TypeScript: Modern UI stack consistent with cockpit-podman.
  - Client wrapper: `cockpit.spawn()` based client for all comquad CLI calls with `--json` flag.
  - Type definitions: TypeScript interfaces matching all Go JSON schemas for type safety.
  - Build system: esbuild-based build with watch mode, ESLint, and Stylelint.

### Changed

- **`comquad list` table format** — Switched from raw printf formatting to go-pretty table (consistent with `view` and `ps` commands). Columns: NAME, STATUS, SERVICES, SOURCE.
- **`comquad list` status logic** — Queries systemd to determine project health based on container unit states. Status values: `healthy` (all running), `degraded` (some running), `stopped` (none running), `up` (no container services).
- **`comquad list` services column** — Shows running/total count (e.g., `2/3`) instead of file count.
- Moved `List()` implementation from `cmd/comquad/list.go` to `internal/orchestrator/list.go` for consistency with other commands.
- Updated README.md command table to reflect new `comquad list [PROJECT]` syntax.
- **SELinux labeling behavior** — `ApplySELinux()` now uses shared volume information to apply `:Z` (private) for single-service volumes and `:z` (shared) for multi-service volumes, instead of always applying `:z`.
- **Dockerfile FROM normalization** — `PatchDockerfileFROM()` now respects `WithoutImageNormalization()` setting, allowing users to skip normalization when respecting `registries.conf`.
- Updated `isSharedVolume()` to handle volume names with project prefix (both `cq-<project>-<name>` and `<project>-<name>` formats).
- Updated orchestrator logging to reflect differentiated SELinux labeling (`:z/:Z` instead of just `:z`).
- Updated documentation across README.md, ARCHITECTURE.md, DEVELOPMENT.md, and compose2quadlet docs to reflect new features and behaviors.

### Fixed

- Volume name extraction in `isSharedVolume()` now correctly handles volume names with project name prefix when `WithoutPrefix()` is used.

### Testing

- Added `TestList_ShowsStatusAndServices` to verify status and service count display.
- Added `TestList_SystemdError` to verify proper error handling when systemd connection fails.
- New tests for `List()` function verifying status display, service counting, and systemd error handling.
- Added `mapper/dockerfile_test.go` with 9 tests for Dockerfile FROM patching and normalization.
- Added `compose2quadlet/shared_volumes_test.go` with 6 tests for shared volume analysis.
- Added integration tests `TestTranspile_SharedVolumesSELinux` and `TestTranspile_SkipImageNormalization` in `transpile_test.go`.
- Added 3 new tests in `mapper/image_test.go` for skip normalization behavior.
- Added 6 new tests in `opinionated/opinionated_test.go` for systemd specifiers and updated SELinux tests.
- New test fixture `testdata/shared-volumes.yaml` for testing shared volume SELinux behavior.
- Added `TestView_ProjectJSON` and `TestView_UnitFileJSON` for view command JSON output.
- Added `TestStart_JSON` for lifecycle command JSON output.
- Added `TestDown_JSON` for down command JSON output.

## [0.4.0] - 2026-09-23

### Added

- **`--json` global flag** — New `--json` global flag for machine-readable output. When enabled, commands output structured JSON with a version field for API stability. Currently supported by `list` and `ps` commands.
- New `internal/output` package providing JSON formatting utilities with versioned API envelope (`{"version": "1.0", "data": {...}}`).
- JSON tags added to `ContainerInfo` and `PortInfo` structs for consistent JSON serialization.
- **X-extension support** — X-extension support for quadlet-only directives that have no Compose equivalent. Users can now access the full Quadlet feature set via compose file extensions:
  - `x-container` for `[Container]` directives (Timezone, Notify, HealthOnFailure, etc.)
  - `x-image` for `[Image]` directives (AllTags, AuthFile, CertDir, etc.)
  - `x-build` for `[Build]` directives (DNS, Environment, Volume, etc.)
  - `x-network` for `[Network]` directives (ContainersConfModule, etc.)
  - `x-volume` for `[Volume]` directives (User, Group, Image, etc.)
  - `x-systemd` for `[Service]` and `[Unit]` directives (MemoryMax, Restart, After, etc.)
  - X-extensions override compose-generated directives when there's a conflict.
  - Unknown directives emit warnings but are passed through for forward compatibility.
  - New `mapper/extensions.go` with extraction functions for all extension types.
  - New `mapper/validation.go` with directive validation against known Quadlet directives.

### Changed

- Refactored `list` command to support both human-readable table output and JSON format.
- Refactored `ps` command to support both human-readable table output and JSON format.
- `ContainerInfo` and `PortInfo` structs now include JSON tags for field mapping.
- Updated `ARCHITECTURE.md` with JSON output section and repository structure.
- Updated `DEVELOPMENT.md` with `internal/output` package documentation.
- Updated `compose2quadlet/README.md` with X-extension documentation and usage examples.
- Updated `compose2quadlet/ARCHITECTURE.md` with X-extension design details and integration points.
- Updated `compose2quadlet/doc/mapping.md` section 16 to mark quadlet-only directives as available via x-extensions.
- Updated `TODO.md` with detailed timer design decisions (deferred to future implementation).
- Modified `mapper/container.go`, `mapper/image.go`, `mapper/build.go`, `mapper/service.go`, `mapper/unit.go`, `mapper/network.go`, and `mapper/volume.go` to integrate x-extension extraction and merging.

## [0.3.5] - 2026-09-21

### Added

- **`comquad build` command** — New `comquad build` command that builds images defined in compose.yaml without starting containers. Supports `--pull always|missing|never` (default: `always`), `-f` to follow build logs, `--dry-run`, and `--no-diff` flags.

### Changed

- **Image pulling delegated to systemd** — Image pulling is now delegated to systemd via `.image` quadlet units instead of direct `podman pull` calls. The `--pull` flag value is applied as `Policy=` on `.image` units, allowing quadlet to enforce pull behavior natively (requires podman >= 5.6). On older podman versions, a verbose-mode warning is emitted.
- Split `startUnits()` into `prepareUnits()` (stops removed units and reloads daemon) and `startContainers()` (starts container units only), with image and build handling as separate phases.
- Added `setPolicyOnImageUnits()` to apply `Policy=` directive on `.image` units based on the `--pull` flag.
- Updated `Up()` lifecycle to: `prepareUnits → handleImages → handleBuilds → startContainers`.
- Updated tests to cover new image/build handling via systemd and the `comquad build` command.
- `comquad --version` now uses `debug.ReadBuildInfo()` to show the correct version when installed via `go install`, instead of always showing "dev".
- `comquad down -d` now displays a warning about permanent volume deletion before the confirmation prompt.
- Added `WithoutServiceName()` option to compose2quadlet library to suppress `ServiceName=` emission on container units, eliminating the need for post-processing `stripServiceName()` in the orchestrator.
- Added warnings when SELinux relabeling is applied to dangerous host directories (`/etc`, `/var`, `/home`, `/usr`, `/sys`, `/proc`, `/boot`, `/root`). The list is configurable via `COMQUAD_SELINUX_DANGEROUS_PATHS` environment variable.

### Fixed

- **SELinux `Mount=` directive format** — Bind mounts in `Mount=` directives were emitting `selinux=z` as a comma-separated option (e.g., `Mount=type=bind,source=/data,destination=/data,selinux=z`), which is not a valid kernel mount option and failed with "selinux: invalid mount option". Now uses `relabel=shared` for `:z` and `relabel=private` for `:Z`, matching the `podman --mount` option format. `Volume=` directives continue to use `,z`/`,Z`. (`compose2quadlet/mapper/container.go`, `compose2quadlet/opinionated/selinux.go`)
- **`comquad view` missing bind-mount volumes** — The `view` command's SERVICES table only parsed `Volume=` directives, ignoring `Mount=` directives from bind mounts. Bind-mount volumes now appear in the VOLUMES column with `source:destination` format. (`internal/orchestrator/view.go`)

### Testing

- **selinux_test.go** — Updated dry-run content checks for `relabel=shared` format on `Mount=` directives and `SELinux detected` verbose log message.
- **view_edit_test.go** — Updated status text checks from `"down"` → `"stopped"` and unit name matching to use short service names.
- **Containerfile** — Bumped base image from `fedora:41` to `fedora:43` to get Go 1.25+ from distro packages, matching the project's `go.mod` requirement.

## [0.3.4] - 2026-08-19

### Fixed

- `--since 10m` is now accepted and normalized to journalctl's relative-time format.
- `view` and `edit` now reject ignored extra positional arguments and document the correct service syntax.
- External network and volume references retain their configured Podman resource names.
- Failed reconcile writes now roll back already-applied target and baseline changes.
- `oom_kill_disable` now maps to `PodmanArgs=--oom-kill-disable` instead of the unrelated `ManagedOOMMemoryPressure=kill` systemd directive.
- Teardown now removes only networks and volumes carrying both comquad management and project labels.
- `exec` honors explicit `ContainerName=` values and keeps stdin open for interactive sessions.
- `ps` derives service names correctly for projects containing hyphens and normalizes inspected container names.
- Services without explicit networks now receive the generated default network even when other user-defined networks exist.
- Secret and Dockerfile cache write failures now produce warnings instead of being silently ignored.

### Changed

- Moved `compose2quadlet` into the root Go module and removed the local module replacement, allowing versioned `go install github.com/Inoriol/comquad/cmd/comquad@latest` installs.
- Updated readme with nicer presentation
- Updated root and compose2quadlet architecture documentation for external resources, default networks, warning reporting, reconciliation rollback, and OOM handling.
- Added focused regression tests for the corrected mappings and runtime behaviors.

## [0.3.3] - 2026-08-18

### Fixed

- `comquad logs -f` no longer passes an empty `--since=` to journalctl when `--since` is not provided, which caused journalctl to fail with "Failed to parse timestamp".
- A failed `comquad up` no longer unconditionally removes the secrets directory during rollback, preserving secrets from prior successful deployments.
- `comquad regenerate` resource count now includes Images and Builds in the displayed total.
- `comquad list` no longer creates an unnecessary Orchestrator (which calls `os.Getwd()`), making it more robust when the working directory is unavailable.

### Changed

- `rollbackDeploy` signature simplified: removed the unused `secretsDir` parameter (4 call sites updated).
- ARCHITECTURE.md clarified default network attachment behavior: any container without a `Network=` directive is auto-attached to `cq-default`, even when user-defined networks exist.
- Corrected misleading comment in `parseContainer` about Podman container name prefix.

## [0.3.2] - 2026-08-17

### Fixed

- `comquad list` no longer filters results by the current directory name; with no `-n` it lists every deployed project.
- File/unit discovery now uses the full `cq-<project>-` prefix (with trailing `-`), so projects sharing a name prefix (e.g. `web` vs `webapp`) no longer cross-contaminate `up`/`down`/`view`/`regenerate`.
- Podman version is now detected and passed to compose2quadlet via `WithPodmanVersion`, so directive gating reflects the installed podman instead of assuming "latest".
- `comquad logs -f` now follows every unit group concurrently (previously only the first running unit's logs were streamed).
- `comquad ps` now merges systemd state using the correct `cq-<project>-<service>.service` unit name.
- `start`/`stop`/`restart` service matching now correctly resolves `.image`/`.build` units by their service-unit name.
- A failed `comquad up` now rolls back only the files and baseline it touched (restoring the previous deployment) instead of tearing the whole project down.
- `comquad regenerate` now populates `resources.images`/`builds` from `.image`/`.build` filenames.

### Changed

- Extracted the failed-deploy rollback into `Orchestrator.rollbackDeploy` for testability.
- Removed dead code (`hasBuildFile`, `resolveContainerImages`, `normalizeImageRef`, `isRegistryWithPort`, `truncate`).
- Clarified ARCHITECTURE.md label documentation: `.image` units carry no `Label=` directive and are identified by filename.

### Added

- Regression tests covering the fixes above (`list` no-filter, `ps` systemd merge, `resolveUnits` image/build matching, failed-deploy rollback, podman version detection).

## [0.3.1] - 2026-08-14

### Added

- **Change detection & reconcile on `comquad up`** — Re-deploying an existing project shows a color-coded unified diff and asks for confirmation before applying.
  - `--no-diff` flag to skip the diff and confirmation.
  - `--dry-run` now renders a diff too: new files as full content, changed files as a diff, and dropped services as removals.
- **New `internal/reconcile` package**:
  - `MergeUnit` — directive-level three-way merge (base / disk / regenerated).
  - `Compute` / `Apply` — read-only planning split from writing, with atomic file writes.
  - Unified-diff rendering with ANSI color (respects `NO_COLOR` and non-TTY stdout).
- Baseline tracking under `$XDG_DATA_HOME/comquad/baseline/<project>/` to power the merge.
- Selective restart: only created units are started and only changed containers/images are restarted; dropped services are stopped before `daemon-reload`.

### Changed

- `comquad up` reconciles instead of blindly overwriting: manual `comquad edit` changes are preserved via the three-way merge (user edit wins on conflict, with a warning).
- `comquad down` now verifies container units are stopped before removing files, and removes the baseline directory.
- `comquad regenerate` clears the baseline for discovered projects (next `up` falls back to a 2-way reconcile).

### Fixed

- Non-deterministic directive ordering in compose2quadlet that produced spurious diffs on re-deploy:
  - `opinionated.ApplyLabels` now sorts injected label keys.
  - `mapper` now sorts `svc.Networks` and `depends_on` map iteration.
- Removed unused (empty) placeholder fields from `compose2quadlet/doc/mapping.md`.

## [0.3.0] - 2026-08-12

### Breaking Changes

- **`podlet` binary no longer required** — Transpilation is now handled by the `compose2quadlet` Go library, compiled directly into the binary. The `podlet` package can be removed from your system if no other tools depend on it.

### Changed

- **Architecture overhaul** — Replaced the 5-stage pipeline (preprocess → podlet transpile → cook → graft → deploy) with a 2-stage pipeline (compose2quadlet TranspileFile → deploy). All compose→quadlet mapping, opinionated transforms, Dockerfile patching, secret resolution, and image/build quadlet generation are now handled by the library.
- Removed packages: `internal/preprocess/`, `internal/transpile/`, `internal/cooker/`, `internal/graft/`
- Added: `internal/deploy/selinux.go` (moved SELinux detection from preprocess)
- Modified: `internal/orchestrator/` (Up, pipeline, images rewritten), `cmd/comquad/check.go` (removed podlet check)
- Removed dependency: `gopkg.in/yaml.v3` (direct)
- Added dependency: `github.com/Inoriol/comquad/compose2quadlet` (in-tree sub-module at `./compose2quadlet/`)
- **compose2quadlet integration** — Moved `compose2quadlet` library into the comquad repository as an in-tree sub-module at `./compose2quadlet/`
  - Module path: `github.com/Inoriol/comquad/compose2quadlet`
  - Remains importable by external projects as a standalone library
  - Uses a `replace` directive in `go.mod` for local resolution

### Fixed

- **Image reference resolution** — `resolveImageRef()` was comparing `unit.Name` (without extension) against the full `.image` filename, causing `podman pull` to attempt pulling `.image` filenames as image names. Fixed by stripping `.image` suffix before lookup.
- **ServiceName unit naming conflict** — compose2quadlet's `ServiceName=` directive (podman ≥5.3) caused systemd to name units after compose service names instead of file-derived names, breaking comquad's `WaitForUnit` lookup. Fixed by stripping `ServiceName=` from container units after transpilation.
- **Missing .image service dependencies** — Containers referencing `.image` files via `Image=` failed to start when quadlet didn't produce `.image` service units. Fixed by resolving `.image` references to actual image names in container files after transpilation, while keeping `.image` files on disk for documentation and auto-update.
- **AutoUpdate + short image names** — `AutoUpdate=registry` on containers requires fully-qualified image references. Fixed by normalizing image names in `.image` units (`mariadb:10.5` → `docker.io/library/mariadb:10.5`) using the existing `normalizeImage()` function from the dockerfile mapper.
- **RetryDelay duration format** — `.image` unit `RetryDelay=` was emitted as raw nanoseconds (`5000000000`) instead of a duration string (`5s`), causing quadlet generator to reject the file and skip producing `.image` service units.
- **Image labels leaking into sections** — `ApplyLabels` was injecting `Label=` into `[Service]`, `[Unit]`, and `[Image]` sections which don't support it. Fixed to only inject labels in `[Container]`, `[Network]`, `[Volume]`, and `[Build]` sections.
- **ContainerName injection** — Added `ApplyContainerName` opinionated transform that injects `ContainerName=<project>-<service>` (e.g. `nextcloud-redis-mariadb-db`) into every container, matching Docker Compose's container naming convention.
- **Exec argument quoting** — Command arguments in `Exec=` directive now use proper systemd quoting (`'--arg' '--arg'`) instead of space-joined unquoted format.
- **Volume references with .volume suffix** — Named compose volumes in `Volume=` directives now use `.volume` suffix references (`db_data.volume:/path`) so the quadlet generator creates proper dependency chains to `.volume` units.
- **NetworkName consistency** — Networks now emit `NetworkName=<canonical-name>` alongside volumes' `VolumeName=`, giving both resource types explicit naming.
- **Network aliases** — `ApplyNetworkAliases` now injects only compose-relevant aliases: `<service>` and `<project>-<service>`. Removed network-name aliases that were not part of the compose service discovery pattern.
- **Reference prefixing for colon-separated values** — `ApplyReferences` now correctly prefixes `Volume=` and `Mount=` values that contain colon-separated paths (e.g. `db_data.volume:/var/lib/mysql` → prefixed correctly).
- **Daemon-reload scope** — `startUnits()` was only passing `.container` files to `ReloadDaemon()`, so `.image` and `.build` quadlet files were never scanned by the generator. Fixed to pass all project files.
- **Port offset logging** — Port offset messages now appear at the action level (visible by default, not requiring `-v`). Added `WithInfo()` callback option to compose2quadlet for consumer logging of internal transformations.
- **Section spacing** — Blank lines are now inserted between non-empty sections in serialized quadlet files, matching the format produced by the previous podlet-based pipeline for readability.
- **Port offset threshold** — Changed from `< 1024` to `≤ 1024`. Port 1024 is now also offset in rootless mode since it is considered an edge-case privileged port in some environments.
- **Missing `ImageTag=` in `.build` units** — Quadlet `.build` files require `ImageTag=` to be present. Without it, the quadlet generator silently skips producing the corresponding `.service` unit, causing deployment timeouts. Now defaults to `<project>_<service>:latest` (Docker Compose naming convention) when no explicit build tags are set.
- **Healthcheck `ExecStartPre` container reference** — `podman healthcheck run` was passed a quadlet file reference (`.container` suffix) instead of the actual Podman container name. Now uses `<project>-<service>` matching the `ContainerName=` directive, so healthcheck polling finds the target container.
- **Dockerfile `FROM` normalization file path** — Patched Dockerfile copies are now written to `~/.cache/comquad/builds/<project>/<service>.Dockerfile` with absolute `File=` references in `.build` quadlet files. This keeps patched files out of the user's source tree. Cleaned up automatically on `down`.
- **`AutoUpdate=registry` on build-backed containers** — `AutoUpdate=registry` is no longer applied to containers that reference `.build` files. Build-backed containers use locally-tagged images (short names like `project_service:latest`) which fail the fully-qualified reference check required by auto-update.
- **Build start timeout** — `unitStartTimeout` increased from 30s to 5 minutes. Build-backed containers take longer to start because systemd resolves the `.build` service dependency before launching the container, and builds (especially Maven/Gradle) can take several minutes.

### Added

- **Dry-run shows patched Dockerfile content** — `comquad up --dry-run` now displays the full patched Dockerfile content for each build service, showing the `FROM` line normalization that would be applied.

### Documentation

- Updated README.md, ARCHITECTURE.md, and ROADMAP.md to reflect the 2-stage pipeline
- Removed podlet from CI workflow and integration test Containerfile
- Added compose2quadlet section to ARCHITECTURE.md with links to sub-project docs
- Updated ARCHITECTURE.md (root and compose2quadlet) to reflect all changes
- Updated compose2quadlet README.md with new opinionated transforms

## [0.2.1] - 2026-08-10

### Fixed

- **`Volume=` reference corruption** — Short unit names matching as substrings inside longer volume/container names caused mangled references (e.g. `db` inside `mariadb` would produce `cq-nextcloud-redis-mariacq-nextcloud-redis-mariadb-db-nc_data.volume`). Applied `valueContainsRef` standalone matching to both `Volume=` and `Network=` directives in the cooker's reference rewriting step.

## [0.2.0] - 2026-08-10

### Added

- **Build support** — `build:` blocks in compose files are now fully supported
  - `build:` is stripped from YAML before podlet transpilation (avoids podlet's fragile build handling)
  - `.build` quadlet files are generated directly by `BuildQuadletHandler` in graft step with proper `ImageTag=`, `File=`, `SetWorkingDirectory=`
  - `.container` files reference `.build` files via `Image=cq-<project>-<service>.build` (per quadlet spec — systemd handles dependency chain)
  - Anonymous builds (no explicit `image:`) automatically get `<project>-<service>:latest` image tag
  - Dockerfile `FROM` lines are patched with `docker.io/library/` prefixes for registry consistency; stage aliases preserved
  - Patched Dockerfiles cached at `$XDG_CACHE_HOME/comquad/builds/<project>/`
  - `AutoUpdate=registry` stripped from built containers (meaningless for local images)
  - `.image` quadlet generation skipped for built containers
  - Registry pulls skipped for built containers (`hasBuildFile` check in `handleImages`)
  - `Containerfile` checked before `Dockerfile` as default build file (Podman convention)
  - Build cache cleaned up on `comquad down`
  - `comquad up -f` starts journal following *before* units start — build output streams live
  - Dry-run shows `[build]` label for built containers

### Fixed

- Cooker reference rewriting: `Network=` values no longer corrupted by substring matches during rename (fixes `Network=cq-cq-nginx-nodejs-redis-nginx-nodejs-cq-...default.network` mangling)
- `comquad ps` COMMAND column capped at 2 lines (60 chars)

### Added

- `internal/preprocess/dockerfile.go` — Dockerfile FROM line patching with stage alias tracking
- `internal/preprocess/dockerfile_test.go` — 16 test cases
- `internal/preprocess/build.go` — `ExtractServiceBuildSpecs`, build path resolution, Containerfile support
- `internal/graft/handlers/build.go` — `BuildQuadletHandler` generates `.build` quadlet files
- `internal/preprocess/models.go` — Added `ServiceBuildSpec` struct
- `internal/preprocess/engine.go` — Strips `build:` from YAML, injects image tags for anonymous builds
- `internal/graft/engine.go` — Wires `BuildQuadletHandler` into the graft pipeline
- `internal/graft/handlers/image.go` — Skips `.image` generation for built containers
- `internal/orchestrator/images.go` — `handleImages` and `printDryRun` skip pulls for built containers
- `internal/orchestrator/orchestrator.go` — `FollowLogs` starts before `startUnits` (live build output); `resolveBuildCacheDir`
- `internal/orchestrator/down.go` — Cleans up build cache on teardown
- `internal/cooker/references.go` — Fixed `Network=` reference rewriting (substring match bug)
- `internal/orchestrator/ps.go` — COMMAND column capped at 2 lines

## [0.1.5] - 2026-08-09

### Added

- **Secrets management** — comquad now intercepts compose `secrets:` and translates them into quadlet-native directives. External secrets become `Secret=` for Podman's native secret store. File-based and environment-based secrets use systemd `LoadCredential=` in `[Service]` sections with `Volume=` mounts to `/run/secrets/<name>` inside containers.
  - New graft handler (`internal/graft/handlers/secret.go`) processes compose secrets
  - New `SecretDef`, `SecretRef`, and `ServiceSecretRefs` types in `internal/preprocess/models.go`
  - New `ExtractSecretSpecs()` function in preprocess extracts secret definitions and per-service references before preprocessing strips them
  - `ComposeFile` struct now includes `Secrets` field to prevent silent dropping of top-level secrets
  - Per-service `secrets:` and top-level `secrets:` stripped from compose YAML before podlet sees it
  - Resolution tree per secret type:
    - `external: true` → `Secret=<name>` in `[Container]`
    - `external: true` + `name:` → `Secret=<alternate_name>`
    - `file: ./path` → `LoadCredential=<name>:<abs_path>` in `[Service]` + `Volume=/run/credentials/<unit>.service/<name>:/run/secrets/<name>:ro`
    - `environment: VAR` → reads `$VAR`, writes to managed file at `$XDG_DATA_HOME/comquad/secrets/<project>/<name>` (0600), same LoadCredential+Volume pattern
  - Custom mount targets supported via compose long syntax (`SecretRef.Target`)
  - `Grafter` struct extended with `SecretsDir` and `DryRun` fields
  - `Orchestrator.Up()` extracts secret specs alongside image specs and passes them through graft
  - `comquad down` removes `$XDG_DATA_HOME/comquad/secrets/<project>/` directory
  - Dry run shows injected directives without writing managed secret files to disk
  - External Podman secrets are never deleted on down (matching docker compose behavior)
- **Simplified secret mounting** — Replaced the `LoadCredential` + `/run/credentials/` indirection with direct `Volume=` bind mounts. The systemd credential approach didn't integrate with quadlet's container lifecycle — podman couldn't find the credentials directory at mount time.
  - File secrets now produce `Volume=<abs_path>:/run/secrets/<name>:ro` directly in `[Container]`
  - Environment secrets write managed file then bind-mount the same way
  - Removed `[Service]` section injection and `insertServiceSection` helper
  - Removed credential path computation (`ContainerFileToUnitName`-based `/run/credentials/...`)

### Changed

- `registerState` now populates the `resources` field in `projects.json` for all five resource types (containers, networks, volumes, images, builds) during `up`, eliminating the previous disparity where only `regenerate` filled this field
- Podman 4.8.0 is now a hard minimum requirement (was 4.4); `comquad up` validates the version on first deploy when no state file exists, and `comquad check` warns on older versions
- `ResourceInfo` struct extended with `images` and `builds` fields
- Updated ARCHITECTURE.md, README.md, handlers/doc.go, and TODO.md to reflect secrets management and other changes

### Fixed

- `comquad up --dry-run` now includes `.image` quadlet files in its preview output. Previously the graft step (which generates `.image` files and rewrites `.container` references) was skipped during dry-run, so `.image` files never appeared. The graft step now runs during dry-run (into the preview temp directory), making dry-run output match what would actually be deployed.
- **Environment secrets now resolve from `.env` files** — `environment:` compose secrets (`environment: "VAR_NAME"`) now check `.env` in the project directory when the OS environment variable isn't set. Resolution order: (1) `os.Getenv("VAR_NAME")` — takes precedence, (2) Parse `.env` from compose project directory — fallback, (3) Empty — warning logged. Added `Content` field to `SecretDef` so values are resolved during extraction.
- **SELinux `:z` labels on secret volumes** — Secret `Volume=` directives now get `:z` appended when SELinux is active, matching the cooker's behavior for regular volumes. Idempotent — skips if already present.
- **Volume= placement within [Container] section** — `insertContainerDirective` now inserts directives directly after the last non-empty line in the `[Container]` section instead of before the next section header. This prevents blank line separation that could cause quadlet to ignore the directive.

### Testing

- 15 new preprocess tests for secret extraction, stripping, and validation
- 12 new graft handler tests for SecretHandler and section injection helpers
- 3 new preprocess tests for `.env` resolution and OS/env precedence
- 3 new graft handler tests for SELinux volume labeling
- Removed `TestInsertServiceSection` and `TestInsertServiceSection_ExistingSection`
- Updated all existing secret handler tests for direct bind mount format
- All existing tests pass; `go vet` and `go test -race` clean

## [0.1.4] - 2026-08-08

### Added

- **`.image` quadlet generation** — comquad now automatically creates `.image` quadlet files for every service container. Image-related compose fields are extracted from the compose spec and moved into dedicated systemd image units, enabling systemd-managed image pulls with auto-update support.
  - New graft handler (`internal/graft/handlers/image.go`) creates `.image` files from `.container` quadlets
  - Compose fields mapped to `.image` quadlet:
    - `image` → `Image=`
    - `pull_policy` → `Policy=` (always, missing, never; `if_not_present` aliased to `missing`)
    - `platform` → `OS=`, `Arch=`, `Variant=` (parsed from `os/arch/variant` format)
  - Defaults: `Retry=3`, `RetryDelay=5s`
  - `.container` files updated to reference `Image=cq-<project>-<service>.image`
  - `pull_policy` and `platform` fields stripped from compose YAML before passing to podlet
  - `handleImages()` resolves `.image` references to real image names during deployment
  - `startUnits()` starts `.image` units before `.container` units, with graceful fallback
- **View/Overview command redesign** — Added `overview` alias for `view` command. Completely redesigned output with two distinct tables:
  - SERVICES: containers with their image, attached networks, and volumes
  - RESOURCES: images, networks, and volumes with copy-pasteable names (e.g. `db.image`)
  - Header now shows Project, Source, Status on three lines
  - Uses `github.com/jedib0t/go-pretty/v6/table` for auto-width columns with clean borders
  - `ps` command also migrated to the same table library

### Changed

- `comquad view` (now `overview`) shows service-to-resource relationships
- Resource names in view output are copy-pasteable (e.g. `comquad view db.image`)
- Tables in both `view` and `ps` use auto-width columns, no more broken alignment
- `ps` COMMAND column capped at 30 chars with automatic text wrapping

### Fixed

- Race condition in `TestTranspile_ReturnsErrorOnNonZeroExit` (broken pipe on stdin write)
- Empty `pull_policy` no longer triggers "unsupported pull_policy" warning
- `.image` quadlet files no longer contain invalid `[Image]` section directives (AutoUpdate, Label)
- Cooker no longer injects `[Image]` section labels or AutoUpdate for `.image` files
- `MatchQuadletResource` now supports short-name matching and service unit name resolution
- `comquad down` now shows the resolved project name in the confirmation prompt

## [0.1.3] - 2026-08-07

### Added

- Full support for `.image` and `.build` quadlet file types across the entire pipeline:
  - Cooker: `[Install]` section, `AutoUpdate`, and project labels injected into `.image`/`.build` files
  - Resolve: `MatchNetworkOrVolume` extended (renamed to `MatchQuadletResource`) to match `.image`/`.build`
  - Lifecycle: `start`, `stop`, `restart` now include `.image`/`.build` alongside `.container` units
  - Down: `.image`/`.build` units are stopped and their files removed during teardown
  - Edit: `resolveAllProjectFiles` includes `.image`/`.build` files
  - Logs: `FollowLogs` streams logs for `.image`/`.build` units
  - Regenerate: `.image`/`.build` files discovered during state reconstruction
  - Deploy: `ReloadDaemon` waits for `.image`/`.build` units to appear after daemon-reload
- New helper functions: `ImageFileToUnitName`, `BuildFileToUnitName`, `fileToUnitName`, `fileToServiceName`

### Changed

- `build:` blocks in compose files are now explicitly rejected by the preprocessor with a clear error message. Build support is not yet implemented.
- `MatchNetworkOrVolume` renamed to `MatchQuadletResource` to reflect expanded scope

### Fixed

- `exec` error message updated from "network or volume" to "non-container unit" to account for `.image`/`.build` types
- ARCHITECTURE.md "Compose Format Implementation Details" corrected — removed 7 fields falsely listed as handled by comquad (`entrypoint`, `command`, `expose`, `deploy`, `environment`, `labels`, `ports`). Consolidated pass-through fields into a single accurate list.
- Documentation updated across README.md, ARCHITECTURE.md, and TODO.md to reflect the above changes.

## [0.1.2] - 2026-08-06

### Fixed

- Numerous fixes related to first ever release in my life.

## [0.1.0] - 2026-08-05

### Added

- All core functions are working! I consider it win.

### Testing

- Added CI pipeline (`.github/workflows/test.yml`): build, vet, short tests, race detector, and coverage on every push and PR to main.
- Added Makefile targets: `build`, `test-unit`, `test-race`, `test-cover`, `test-short`.
- Integration tests now use a pre-built host binary instead of rebuilding inside the container each run.
- Fixed Containerfile: changed `fedora:43` to `fedora:41` (stable); replaced `loginctl enable-linger || true` with direct file creation to avoid silently masked failures.
- Made `captureStdout` goroutine-safe with a `sync.Mutex`, allowing `t.Parallel()` in orchestrator unit tests.

[Unreleased]: https://github.com/Inoriol/comquad/compare/v0.4.4...HEAD
[0.4.4]: https://github.com/Inoriol/comquad/compare/v0.4.4...v0.5.0
[0.4.4]: https://github.com/Inoriol/comquad/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/Inoriol/comquad/compare/v0.4.1...v0.4.3
[0.4.1]: https://github.com/Inoriol/comquad/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/Inoriol/comquad/compare/v0.3.5...v0.4.0
[0.3.5]: https://github.com/Inoriol/comquad/compare/v0.3.4...v0.3.5
[0.3.4]: https://github.com/Inoriol/comquad/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/Inoriol/comquad/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/Inoriol/comquad/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/Inoriol/comquad/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/Inoriol/comquad/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/Inoriol/comquad/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Inoriol/comquad/compare/v0.1.5...v0.2.0
[0.1.5]: https://github.com/Inoriol/comquad/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/Inoriol/comquad/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/Inoriol/comquad/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/Inoriol/comquad/compare/v0.1.0...v0.1.2
[0.1.0]: https://github.com/Inoriol/comquad/releases/tag/v0.1.0
