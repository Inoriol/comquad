# Development Reference

This document contains implementation details that are useful when changing comquad but are not needed to understand its high-level architecture.

## Package Responsibilities

### `cmd/comquad`

Defines the Cobra commands and their flags. Commands construct an orchestrator and pass command-specific options to it. The root command owns the persistent `--quiet`, `--verbose`, and `--json` flags.

### `compose2quadlet`

Loads Compose files through compose-go/v2, maps Compose resources to structured Quadlet units, applies conversion rules, and serializes the result. See its [architecture guide](./compose2quadlet/ARCHITECTURE.md) for package-level details.

### `internal/orchestrator`

Coordinates the deployment pipeline and implements project operations. The main responsibilities are:

- Reading the Compose file and invoking transpilation
- Computing and applying reconciliation plans
- Setting `Policy=` on `.image` units based on `--pull` flag
- Starting `.image` units via systemd to pull images
- Starting `.build` units via systemd to build images
- Starting, stopping, and restarting container units
- Resolving service names for command arguments
- Implementing logs, exec, view, edit, and project recovery

### `internal/reconcile`

Owns change detection independently from deployment side effects:

- `Compute` creates a read-only plan.
- `MergeUnit` performs directive-level three-way merging.
- `Apply` writes files, updates baselines, removes stale files, and rolls back touched files when possible.
- Diff rendering turns plans into human-readable output.

The plan/apply split is important: dry runs and confirmation prompts must be able to inspect changes without modifying the host.

### `internal/deploy`

Provides systemd D-Bus communication, Podman queries, target-directory selection, and project state storage. It also defines the interfaces used to replace external services in tests:

- `SystemdClient` abstracts the D-Bus operations used by the orchestrator.
- `StateStore` abstracts project state operations.

### `internal/logger`

Implements normal, verbose, quiet, and error output. `NO_COLOR` disables ANSI color output. Errors always go to stderr; quiet mode suppresses non-error output.

### `internal/output`

Provides JSON output formatting for machine-readable command output. Key components:

- `schemas.go` — JSON schema definitions with versioned API envelope
- `formatter.go` — `PrintJSON()`, `PrintError()`, and mode management via `SetJSONMode()`/`IsJSONMode()`

To add JSON support to a command:

1. Define the output struct in `schemas.go` (e.g., `ViewData`)
2. In the command or orchestrator method, check `output.IsJSONMode()`
3. Convert internal types to the JSON schema type
4. Call `output.PrintJSON(&output.Envelope{Version: output.APIVersion, Data: yourData})`

The `--json` flag is set in `PersistentPreRun` of the root command, so it is available to all subcommands.

## Compose-to-Quadlet Details

Most conversion behavior belongs in `compose2quadlet`. The orchestration layer performs only the post-processing needed to integrate generated units with comquad's naming and lifecycle model.

The `WithoutServiceName()` option suppresses `ServiceName=` on container units so systemd uses the generated filename, such as `cq-myproject-web.service`, instead of naming the unit only after the Compose service.

The conversion layer also:

- Defaults container names to `<project>-<service>` unless `container_name` is set.
- Adds both service and project-qualified network aliases.
- Keeps `.image` references in container units so Quadlet can build dependencies.
- Preserves external network and volume names.
- Normalizes unqualified images through compose-go/v2. Image normalization can be disabled via `COMQUAD_SKIP_REGISTRY_NORMALIZATION` to respect `registries.conf`.
- Applies differentiated SELinux mount handling: `:Z` (private) for single-service volumes, `:z` (shared) for volumes used by multiple services.
- Optionally shortens paths using systemd specifiers (`%h` for `$HOME`) when `COMQUAD_SYSTEMD_SPECIFIERS` is set.
- Offsets privileged rootless ports using `ROOTLESS_PORT_OFFSET` and resolves internal port conflicts.
- Translates Compose secrets into native Podman secrets or `/run/secrets/<name>` mounts.

For exact field support and generated directives, use the [mapping reference](./compose2quadlet/doc/mapping.md) rather than duplicating that table here.

## Image and Build Units

Each service image is represented by an `.image` unit when applicable. Compose `image`, `pull_policy`, and `platform` values become `[Image]` directives. Containers refer to these units instead of duplicating the image configuration.

Compose `build:` blocks produce `.build` units. Dockerfile `FROM` references are adjusted where needed so locally built images and generated references remain consistent.

Image pulling is delegated to systemd via `.image` quadlet units. The `--pull` flag value (`always`/`missing`/`never`) is applied as `Policy=` on `.image` units during the `up` lifecycle, allowing quadlet to enforce the pull behavior natively (requires podman >= 5.6). On older podman versions, a verbose-mode warning is emitted and the `Policy=` directive is ignored.

The `comquad build` command provides explicit control over image building. It transpiles the compose file, reconciles quadlet files, and starts `.build` units via systemd without starting `.container` units. The default pull strategy is `always` to ensure base images are re-pulled during builds.

## Service Resolution

Commands accepting a service argument share matching logic. A service can generally be resolved by:

- Compose service name
- Quadlet filename or filename without its extension
- Generated systemd service name
- Short name after removing the `cq-<project>-` prefix
- Explicit `ContainerName=` value

`MatchAllContainers` returns all matching containers for commands that support multiple services. `exec` reports an error when a name is ambiguous because it requires exactly one container.

The `ensureProjectDeployed` helper centralizes state lookup for commands that require an existing deployed project.

## Logs and Status

`ps` queries Podman for managed containers and combines the result with systemd unit state. Podman supplies container metadata such as image, command, ports, and exit code; systemd supplies the current unit state.

For running units, `logs` uses the unit's current invocation ID to avoid mixing logs from older container runs. For stopped or failed units, it shows historical journal output. Logs from multiple units are collected as JSON, sorted by timestamp, and prefixed with the unit name.

## Output Levels

| Mode | Flag | Output |
|---|---|---|
| Normal | None | Operational messages, actions, warnings, and errors |
| Verbose | `-v`, `--verbose` | Normal output plus pipeline and transformation details |
| Quiet | `-q`, `--quiet` | Errors only |

Quiet mode takes precedence over verbose mode. Errors are not suppressed.

## Testing Architecture

The orchestrator uses injectable dependencies instead of constructing external clients inside every operation. Its factories include:

```go
newState       func() (deploy.StateStore, error)
newSystemd     func() (deploy.SystemdClient, error)
listContainers func(projectName string, all bool) ([]ContainerInfo, error)
newJournalCmd  func(name string, args ...string) *exec.Cmd
```

Unit tests replace these factories with in-memory state and systemd fakes. This allows command and reconciliation behavior to be tested without a live D-Bus session or Podman daemon.

Integration tests exercise the full Compose-to-running-container path with real Podman and systemd. They run in a privileged test container with systemd as PID 1, an isolated D-Bus session, and a pre-built comquad binary mounted from the host.

The integration test helpers cover:

- Running the CLI and capturing output
- Writing temporary Compose files
- Inspecting Podman containers, networks, and volumes
- Polling systemd unit state
- Reading project state
- Detecting SELinux support

## Local Verification

The CI workflow runs build, vet, unit, race, and coverage checks. Equivalent Make targets are available:

```bash
make test-unit
make test-short
make test-race
make test-cover
```

The basic project checks are:

```bash
go build ./...
go vet ./...
go test -short ./...
```

Integration tests require Podman, systemd, and the test container setup described in `tests/integration/Containerfile`.
