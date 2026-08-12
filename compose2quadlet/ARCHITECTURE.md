# Architecture Guide

This document describes the internal design, conventions, and data flow of compose2quadlet. It is written for AI agents and new contributors reading the codebase for the first time. Keep it updated as the project evolves.

## Official Specification Links

| Spec | URL |
|---|---|
| **Compose Specification** | https://github.com/compose-spec/compose-spec/blob/master/spec.md |
| **Compose Build Spec** | https://github.com/compose-spec/compose-spec/blob/master/build.md |
| **Compose Deploy Spec** | https://github.com/compose-spec/compose-spec/blob/master/deploy.md |
| **Quadlet (podman-systemd.unit.5)** | https://github.com/podman-container-tools/podman/blob/main/docs/source/markdown/podman-systemd.unit.5.md |
| **systemd.unit** | https://github.com/systemd/systemd/blob/main/man/systemd.unit.xml |
| **systemd.service** | https://github.com/systemd/systemd/blob/main/man/systemd.service.xml |
| **systemd.resource-control** | https://github.com/systemd/systemd/blob/main/man/systemd.resource-control.xml |
| **systemd.exec** | https://github.com/systemd/systemd/blob/main/man/systemd.exec.xml |
| **compose-go (Go types)** | https://github.com/compose-spec/compose-go |
| **Podman Release Notes up to v5.6.0** | https://github.com/podman-container-tools/podman/blob/main/RELEASE_NOTES.md |
| **Podman Release Notes after v5.6.0** | https://github.com/podman-container-tools/podman/releases |

## Version Tracking

Every compose→quadlet mapping is tracked with a minimum podman/systemd version in `doc/mapping.md`.
The `Since` column records when the target directive was introduced.

**Minimum baseline: Podman 4.8.0** (required for `.image` quadlet support).
Quadlet types by introduction: `.container`/`.network`/`.volume` 4.4.0, `.image` 4.8.0, `.build` 5.2.0.

When adding a new field mapping, check the [Podman Release Notes](https://github.com/podman-container-tools/podman/blob/main/RELEASE_NOTES.md)
to determine the minimum podman version. For systemd directives, the version is noted in the `Added in version X`
footer of each option in `systemd.resource-control`.

## Project Scope

**Library, not a CLI.** No `main()`, no cobra, no command-line interface. Two entry points: `Transpile()` for callers who already have a compose-go `*types.Project`, and `TranspileFile()` which loads the compose file and transpiles in one call.

**Narrow focus:** compose → quadlet only. No Kubernetes, no pods, no artifacts. Only quadlet types relevant to compose: `.container`, `.network`, `.volume`, `.image`, `.build`.

**Consumer:** [comquad](https://github.com/Inoriol/comquad) imports this library. comquad keeps the CLI, orchestration (`up`/`down`/`start`/`stop`/`logs`), state management, and D-Bus communication. The library absorbs the `internal/{preprocess,transpile,cook,graft}` pipeline.

## Package Structure

```
compose2quadlet/
├── ARCHITECTURE.md           # This file
├── README.md                 # End-user documentation
├── go.mod                    # module github.com/Inoriol/comquad/compose2quadlet
│
├── TODO.md                   # Known issues and planned improvements
├── types.go                  # Type aliases re-exporting from internal/types/
├── transpile.go              # Entry points: Transpile(project, opts...), TranspileFile(path, opts...)
├── options.go                # TranspileOption constructors, delegates to internal/types/
├── helpers_test.go           # Shared test helpers for root-level tests
├── version_test.go           # Tier 2 — version matrix tests
├── transpile_test.go         # Tier 3 — pipeline integration tests
│
├── testdata/                 # Test fixtures
│   ├── simple-web.yaml
│   ├── multi-service.yaml
│   ├── edge-cases.yaml
│   ├── top-level.yaml
│   ├── version-basic.yaml
│   ├── version-build.yaml
│   ├── version-features.yaml
│   ├── version-memory.yaml
│   └── serialization/        # Tier 0 — golden files
│       ├── container.golden
│       ├── network.golden
│       ├── volume.golden
│       ├── image.golden
│       └── build.golden
│
├── internal/
│   └── types/                # Shared types — no import cycles
│       ├── core.go           # QuadletUnit, Section, Directive, Warning, WarningLevel, UnitType, section constants
│       └── config.go         # Config, Version, Option, DefaultConfig(), Warn()
│
├── mapper/                   # Field mapping logic
│   ├── container.go          # Container(), t0Container(), t1Container(), t3Container() — P1 [Container] directives
│   ├── service.go            # Service() — P2 [Service] directives (memory, CPU, IO, restart, deploy)
│   ├── unit.go               # Unit(), UnitService() — depends_on → [Unit] deps + health polling
│   ├── healthcheck.go        # Healthcheck() — healthcheck directives
│   ├── security.go           # SecurityOpts() — security_opt parsing
│   ├── ports.go              # formatPort() helper
│   ├── image.go              # Images() — .image companion quadlets
│   ├── build.go              # Builds() — .build quadlets (fatal error pre-5.2.0)
│   ├── network.go            # Networks() — top-level .network quadlets
│   ├── volume.go             # Volumes() — top-level .volume quadlets
│   ├── secrets.go            # PremapSecrets() — secrets/configs pre-mapping interceptor (env, file, external)
│   ├── dockerfile.go          # PatchDockerfileFROM() — normalize bare image names in FROM lines
│   └── helpers.go             # sortedKeys() — deterministic map key ordering
│
├── opinionated/              # Opinionated transforms
│   ├── opinionated.go        # Apply() — orchestrates all transforms
│   ├── prefix.go             # ApplyPrefix() — cq-<project>- prefix on all unit names
│   ├── references.go         # ApplyReferences() — rewrite Network=, Volume=, Image=, Mount= references
│   ├── containername.go      # ApplyContainerName() — inject ContainerName=<project>-<service>
│   ├── aliases.go            # ApplyNetworkAliases() — inject NetworkAlias=<service>, <project>-<service>
│   ├── selinux.go            # ApplySELinux() — add relabel=shared to Mount=, :z to Volume=
│   ├── labels.go             # ApplyLabels() — inject consumer-provided labels (Container/Network/Volume/Build only)
│   ├── network.go            # ApplyDefaultNetwork() — inject default network if needed
│   ├── ports.go              # ApplyPortOffset() — apply port offset with Info callback logging
│   ├── autoupdate.go         # ApplyAutoUpdate() — add AutoUpdate=registry
│   └── install.go            # ApplyInstallSection() — add [Install] section
│
├── serialization/            # Serialization / deserialization
│   └── ini.go                # Marshal(), Write(), WriteUnits(), Unmarshal()
│
└── doc/
    └── mapping.md            # Complete field-by-field mapping reference
```

## Core Types

```go
// UnitType identifies the kind of quadlet unit file.
type UnitType string  // "container" | "network" | "volume" | "image" | "build"

// QuadletUnit is a structured representation of a quadlet unit file
// before serialization to ini-format.
type QuadletUnit struct {
    Type     UnitType
    Name     string     // base name, before prefixing (e.g. "web", "db")
    Sections []Section
}

// Section represents an ini section like [Container], [Service], [Unit].
type Section struct {
    Name       string     // e.g. "Container", "Service", "Unit", "Install"
    Directives []Directive
}

// Directive is a single key-value entry. Values stores multiple
// values for directives that repeat on separate lines with the same key
// (e.g. Environment=, Volume=, PublishPort=).
type Directive struct {
    Key    string
    Values []string  // empty = key present with no value (boolean flag)
}
```

### Section Constants

Predefined section names used across the codebase:
```
SectionUnit      = "Unit"
SectionService   = "Service"
SectionInstall   = "Install"
SectionContainer = "Container"
SectionNetwork   = "Network"
SectionVolume    = "Volume"
SectionImage     = "Image"
SectionBuild     = "Build"
```

## Data Flow

```
compose.yaml
    │
    ▼
compose-go/loader.Load()  ← handles interpolation, env resolution, extension merging
    │
    ▼
*types.Project   ← canonical, fully-resolved compose model
    │
    ▼
Transpile(project, opts...)
    │
    ├── 1. Apply transpileConfig (defaults + user overrides from opts)
    │
    ├── 2. Secrets pre-mapping intercept
    │       secrets → Volume= / Secret= in [Container]
    │       configs → Mount=type=bind in [Container]
    │       Environment-based secrets resolved when WithSecretsDirectory() is set
    │       (strips secrets/configs from model before field mapping)
    │
    ├── 3. Field mapping phase (mapper/)
    │       For each service:
    │       ├── container.go: service fields → [Container] directives (P1/P3) ✅
    │       ├── unit.go: depends_on → [Unit] After=/Requires= + health polling ✅
    │       ├── healthcheck.go: healthcheck → HealthCmd= etc. ✅
    │       ├── service.go:  systemd resources/restart → [Service] directives (P2) ✅
    │       ├── image.go:    image → .image quadlet ✅
    │       └── build.go:    build → .build quadlet (with optional Dockerfile FROM normalization) ✅
    │
    ├── 4. Top-level networks/volumes → .network/.volume quadlets ✅
    │       ├── network.go:  project.Networks → .network units ✅
    │       └── volume.go:   project.Volumes → .volume units ✅
    │
    ├── 5. Opinionated transforms (opinionated/) ✅
    │       ├── prefix.go:        cq-<project>- prefix on all unit names
    │       ├── references.go:    rewrite Network=, Volume=, Image=, Mount=, After= references
    │       ├── containername.go: inject ContainerName=<project>-<service>
    │       ├── aliases.go:       inject NetworkAlias=<service> (service name + project-service)
    │       ├── selinux.go:       add relabel=shared to Mount=, :z to Volume=
    │       ├── labels.go:        inject consumer-provided labels (skips [Service], [Unit], [Image])
    │       ├── network.go:       inject default network if needed
    │       ├── ports.go:         apply port offset
    │       ├── autoupdate.go:    add AutoUpdate=registry
    │       └── install.go:       add [Install] section
    │
    ▼
[]QuadletUnit   ← structured, typed output
    │
    ▼
serialization/ini.go    ← serialize to ini text format (optional; comquad may serialize itself)
    │
    ▼
foo.container, bar.network, baz.volume files written to disk
```

## Field Mapping Priority System

Every compose field maps to exactly one of four levels. This is documented exhaustively in `doc/mapping.md`.

| P | Name | Target | Example |
|---|---|---|---|
| **1** | Direct Quadlet | `[Container]`, `[Network]`, `[Volume]`, `[Image]`, `[Build]` | `ports` → `PublishPort=` |
| **2** | Systemd | `[Service]` (resource-control, restart) or `[Unit]` (deps) | `mem_limit` → `MemoryMax=` |
| **3** | PodmanArgs | `PodmanArgs=` in `[Container]` | `tty` → `PodmanArgs=--tty` |
| **4** | Unsupported | Ignored (or warned) | `deploy.replicas`, `extends` |
| — | Structural | Generates separate quadlet unit | `build` → `.build` unit |

Priority 2 fields go in the `[Service]` section of the container unit. Quadlet passes `[Service]` directives through to the generated `.service` file, so systemd enforces them at the cgroup level. This is superior to using `PodmanArgs` because systemd can enforce limits even if podman is bypassed.

Where both P1 and P2 directives exist for the same field (e.g. `ulimits` → `Ulimit=` P1 vs `LimitXXX=` P2), only P2 is emitted. The systemd enforcement path is always preferred over the equivalent quadlet directive.

## Version Awareness

The library tracks a target podman version (via `WithPodmanVersion()` or defaults to "latest"). Mappers gate output based on this version:

| Target version | `entrypoint:` behavior | `build:` behavior |
|---|---|---|
| Zero / latest | Emit `Entrypoint=` (P1) | Emit `.build` unit (P1) |
| `5.0.0` | Emit `Entrypoint=` (P1, since 5.0) | Emit `.build` unit (P1, since 5.2) |
| `4.8.0` | Emit `PodmanArgs=--entrypoint ...` (P3 fallback) | **Fatal error** — impossible |

The version check is centralized:

- **Inline in mapper functions** — Each field checks `cfg.PodmanVersion.AtLeast(...)` directly. Simple 1:1 fields use a version gate to emit the directive or a warning. Complex fields (e.g. `build` which produces a whole unit, `entrypoint` which has a non-trivial P3 format) emit different directive keys depending on the version.

Systemd versions are tracked only as documentation in `doc/mapping.md`. In practice, modern podman implies modern systemd, so the library collapses to a single podman version axis.

## Warning System

Every field that cannot be mapped is surfaced — there are **no silent skips**. Three severity levels:

| Level | Meaning | Example | Consumer impact |
|---|---|---|---|
| `WarningSkipped` | Feature unavailable at target podman version | `network_aliases` on podman 4.8.0 | Info: "field skipped, requires podman 5.2.0" |
| `WarningDegraded` | P3 PodmanArgs fallback instead of P1 | `entrypoint` on podman 4.8.0 | Warn: "mapped via PodmanArgs, upgrade to podman 5.0 for native support" |
| `WarningFatal` | Mapping is impossible at this version | `build:` on podman 4.8.0 | `Transpile()` returns error |

Warnings are collected in `transpileConfig.Warnings` during the pipeline and surfaced alongside the result. Consumers (comquad) decide how to present each level. No separate error/warning channel is threaded through mapper signatures — the config acts as a shared collector.

```go
// Internal usage in a mapper:
cfg.warn(Warning{
    Level:   WarningDegraded,
    Service:  svc.Name,
    Field:    "entrypoint",
    Message:  "using PodmanArgs fallback",
    Since:    "5.0.0",
})
```

## Opinionated Defaults

All transforms are **enabled by default** and can be individually disabled via `TranspileOption`. This matches comquad's current behavior.

| Transform | Option to disable | What it does |
|---|---|---|
| File prefixing | `WithoutPrefix()` | Prepends `cq-<project>-` to unit filenames |
| Reference rewriting | *(always on)* | Rewrites `Network=`, `Volume=`, `Image=`, `After=`, `Requires=` to prefixed names; handles colon-separated values (e.g. `name.volume:/path`) |
| ContainerName injection | *(always on)* | Adds `ContainerName=<project>-<service>` to containers (e.g. `nextcloud-redis-mariadb-db`) |
| NetworkAlias injection | `WithoutNetworkAliases()` | Adds `NetworkAlias=<service>` and `NetworkAlias=<project>-<service>` for DNS-based service discovery |
| SELinux labeling | `WithoutSELinux()` | Appends `relabel=shared` to `Mount=` bind-mount directives and `,z` to `Volume=` directives |
| Managed label | `WithLabels(map)` | Adds consumer-provided labels to `[Container]`, `[Network]`, `[Volume]`, `[Build]` sections (not `[Service]`, `[Unit]`, `[Image]` which don't support `Label=`) |
| Project label | `WithLabels(map)` + `WithProjectName` | Adds consumer-provided labels to every unit |
| Default network | `WithoutDefaultNetwork()` | Injects `cq-default.network` if no networks defined |
| Port offset | `WithPortOffset(N)` | Adds offset to host-side published ports ≤ 1024; logs changes via `Info` callback |
| AutoUpdate | `WithAutoUpdate()` | Adds `AutoUpdate=registry` to containers |
| Install section | `WithoutInstallSection()` | Adds `[Install] WantedBy=default.target` |
| Image retry | `WithImageRetry(N)` / `WithImageRetryDelay(S)` | Sets `Retry=`/`RetryDelay=` on `.image` units (default: 3 / 5s; RetryDelay uses duration string format e.g. `5s`) |
| Image normalization | *(always on)* | Normalizes bare image names (`nginx:latest` → `docker.io/library/nginx:latest`) in `.image` units |
| Working directory | `WithWorkingDirectory(path)` | Resolves relative bind-mount volume paths against this directory |
| Secrets directory | `WithSecretsDirectory(path)` | Enables environment-based secret resolution; writes managed files to this dir |
| Dry run | `WithDryRun()` | Skips writing managed secret files to disk (still generates directives) |
| Dockerfile normalization | `WithDockerfileNormalization()` + `WithBuildCacheDir(path)` | Patches Dockerfile FROM lines to fully-qualified image names; writes patched copies to cache dir |
| Info callback | `WithInfo(fn)` | Receives info-level messages (e.g. port offset changes) for consumer logging |
| Version parsing | `ParseVersion(s)` | Parses `"5.2.0"` or `"v4.8"` into `Version` for `WithPodmanVersion()` |
| Batch write | `serialization.WriteUnits(dir, units)` | Writes all units to a directory with `<name>.<type>` filenames |

## Quadlet-Specific Behaviors

### Image Quadlet Generation

Every service with `image:` gets a companion `.image` quadlet. The container unit references it via `Image=<name>.image`. This splits image pulling into a separate systemd unit, enabling dependency ordering.

### Quadlet Cross-Reference Syntax

Quadlet uses special extension syntax for unit references:
- `Network=<name>.network` — references a `.network` quadlet
- `Volume=<name>.volume` — references a `.volume` quadlet
- `Image=<name>.image` — references a `.image` quadlet
- `Image=<name>.build` — references a `.build` quadlet

The mapper must output these `.network`/`.volume`/`.image`/`.build` suffixes so that systemd dependency chains are created automatically.

### Dependency Translation

`[Unit]` directives (`After=`, `Requires=`, `Wants=`, `BindsTo=`, `PartOf=`) between quadlet units are automatically translated by the quadlet generator. For example, `After=db.container` in a `web.container` unit creates a proper systemd `After=db.service` dependency.

### Build ImageTag Defaulting

`.build` quadlet files require `ImageTag=` to be present. The `Builds()` mapper emits `ImageTag=` from `build.Tags`; if no tags are set, it defaults to `<project>_<service>:latest` (Docker Compose naming convention). Without this default, the quadlet generator silently skips `.build` files and produces no service unit.

### Dockerfile FROM Normalization

Podman resolves bare image names differently from Docker. When `WithDockerfileNormalization()` and `WithBuildCacheDir(path)` are set, `Builds()` calls `PatchDockerfileFROM()` to normalize `FROM` lines:
- `nginx:alpine` → `docker.io/library/nginx:alpine`
- `library/redis` → `docker.io/library/redis`
- Multi-stage build aliases are tracked and not normalized
- `--platform` flags are preserved
- `FROM scratch` is never normalized
- Patched copies are written to `BuildCacheDir` as absolute paths before the `.build` quadlet is emitted
- During dry-run, content is computed but not written to disk

### Environment-Based Secret Resolution

When `WithSecretsDirectory(path)` is set, `PremapSecrets()` resolves `secrets:` entries that reference environment variables:
- Reads the env var via `os.Getenv(def.Environment)`
- Writes the value to `path/<secret_name>` with `0600` permissions
- Emits `Volume=<path>:/run/secrets/<name>:ro` in `[Container]`
- `WithDryRun()` skips disk writes but still generates correct directives

## Conventions

### Go Code Style
- Standard library only (plus compose-go/v2). No external process execution, no additional YAML libraries.
- Package name: `compose2quadlet` (not `main` — this is a library).
- No code comments in implementation files unless the logic is non-obvious. Mapping is self-documenting via directive names.
- Tests live in `*_test.go` files alongside the code they test.

### Naming
- `QuadletUnit.Name` is the **base name** before prefixing (e.g. `"web"`, `"db"`, `"default"`).
- The full filename is constructed as `<prefix><Name>.<type>` during serialization.
- Section names are PascalCase strings matching ini section headers: `"Container"`, `"Service"`, `"Unit"`.
- Directive keys are the exact quadlet/systemd directive names: `"PublishPort="`, `"MemoryMax="`.

### Directive Ordering
Directives within a section should follow the order from the quadlet spec where possible. Systemd `[Service]` directives go after all `[Container]` directives. `[Unit]` goes before `[Container]`, `[Install]` goes last.

### Deterministic Map Iteration
All map iteration (labels, annotations, environment, sysctls, logging options, extra hosts, ulimits, build args, driver opts, storage opts) must use the `sortedKeys()` helper from `mapper/helpers.go`. This ensures reproducible directive ordering and stable golden file tests.

## Testing Strategy

### Tier 0 — Serialization (`serialization/ini.go`)
QuadletUnit → ini text correctness:

- Section ordering (`[Unit]` → `[Container]` → `[Service]` → `[Install]`)
- Multi-value directive rendering (multiple `Volume=`, `Environment=` lines)
- Empty-value directives (boolean flags: `NoNewPrivileges=`)
- **Empty-default**: each unit type with only mandatory fields renders correctly (no trailing empty lines, no missing `\n`)
- **Empty section omission**: sections with zero directives are not rendered (no dangling `[Unit]\n` header)
- **Round-trip**: serialize → deserialize → identical `QuadletUnit`, for every unit type
- **Comment line**: `# FileName=<name>` header line present/absent in output

### Tier 1 — Mapper Unit Tests (`mapper/*_test.go`)
Each mapper function tested independently with table-driven tests.
Input: compose-go `types.ServiceConfig` (or relevant field subset).
Output: expected `[]Directive` (or `[]QuadletUnit` for structural mappers).

Tests focus on **correct output at latest podman version**. Version-gated behavior is tested in Tier 2.

### Opinionated Transform Tests (`opinionated/*_test.go`)
Each transform tested independently with input `[]QuadletUnit` slices and config, verifying correct mutations to unit names, directives, and sections. Covers all transforms: prefix, references, aliases, SELinux, labels, default network, port offset, auto-update, install section, and the full `Apply()` pipeline.

### Tier 2 — Version Matrix
A dedicated test file (`version_test.go`) that runs the same compose input through multiple target podman versions and asserts correct behavior per version boundary (4.8.0, 5.0.0, 5.2.0, 5.3.0, 5.5.0). Covers:

- Fields that promote from P3 PodmanArgs to P1 native directive (`entrypoint`, `stop_signal`, `extra_hosts`)
- Fields that become available at a boundary (`network_aliases`, `addhost`, `log_options`)
- Fields that switch section (`mem_limit`: `[Service] MemoryMax=` → `[Container] Memory=`)
- Structural blocks that cause fatal errors (`build` on < 5.2.0)
- Correct `Warning` collection at each severity level

### Tier 3 — Pipeline Integration
Full compose YAML → compose-go parse → `Transpile()` → verify `[]QuadletUnit` structure (`transpile_test.go`, `testdata/` fixtures):
- Single service, multi-service, no-service (top-level networks/volumes only)
- Option combinatorics: pairs of enable/disable on opinionated transforms
- Warning collection verification
- Edge cases: build-only service, external volumes/networks

```go
func TestTranspile_SimpleWeb(t *testing.T) {
    project := loadProject(t, "testdata/simple-web.yaml")
    units, err := Transpile(project, WithProjectName("test"),
        WithoutPrefix(), WithoutDefaultNetwork(), WithoutSELinux(),
        WithoutNetworkAliases(), WithoutInstallSection(),
    )
    if err != nil { t.Fatal(err) }
    unit, ok := findUnit(units, "test-web", UnitContainer)
    if !ok { t.Fatal("expected test-web.container unit") }
    sec, ok := hasSection(unit, SectionContainer)
    if !ok { t.Fatal("expected [Container] section") }
    assertDirectiveValue(t, sec.Directives, "PublishPort", "8080:80")
}
```

### Tier 4 — End-to-End (deferred to comquad)
comquad's existing `tests/integration/` harness. The library itself does not start podman or systemd.

### Test Conventions
- Fixture compose files live in `testdata/` at the package root.
- Table-driven tests use `t.Run()` for each entry with descriptive names.
- Golden files for serialization live in `testdata/serialization/` with `.golden` extension.
- Test helper functions (`loadProject`, `findUnit`, `hasSection`, `hasDirectiveValue`) are shared in `helpers_test.go` at the package root.
- No external test dependencies beyond the standard library and compose-go/v2.
- **Empty-default pattern**: every unit type gets a test verifying that a `QuadletUnit` with only mandatory fields serializes correctly — catches section rendering bugs early.
- **Round-trip pattern**: for serialization, every test verifying serialization should also verify deserialization produces the same struct.

## Development Order (Milestones)

From the project scope document:

1. **MVP** — `.container` files only, priority-1 field mappings, no opinionated transforms ✅
2. **Full compose parity** — `.network`, `.volume`, `.image`, `.build` support, all priority-1 + priority-2 ✅
3. **Opinionated defaults** — all comquad transforms ported as opt-out `TranspileOption`s ✅
4. **Deploy + systemd** — `deploy.resources`, `deploy.restart_policy` mapped to `[Service]` ✅
5. **Secrets + builds** — compose `secrets:` and `build:` handled natively ✅
6. **Integration** — comquad imports the library, drops podlet dependency 🔜
7. **Deprecate podlet** — comquad no longer requires podlet binary at runtime 🔜

## Key Design Decisions

### Why structured output instead of text?
Podlet emits ini text. Manipulating text is fragile (regex, line parsing). `QuadletUnit` structs allow programmatic modification — rename files, rewrite references, inject labels, offset ports — before serialization. This eliminates comquad's entire strip/cook/graft pipeline.

### Why not use quadlet directives for everything?
Some compose fields have better systemd equivalents. For example, `mem_limit` could be `PodmanArgs=--memory ...` (P3) but `MemoryMax=` in `[Service]` (P2) is enforced at the cgroup level by systemd itself. Priority 2 always wins over priority 3 when both are possible.

### Why separate `.image` quadlets?
Splitting image pulls into a separate unit enables proper dependency ordering. The `.image` unit completes before the `.container` unit starts. This also enables `AutoUpdate=registry` on the image unit, triggering updates independently.

### Why pre-mapping intercept for secrets?
Secrets and configs need different handling depending on their type (external vs file vs environment). Intercepting them before the field mapper runs avoids collision with the volume mapper and ensures correct `Secret=` vs `Volume=` routing.

### Why normalize Dockerfile FROM lines?
Podman resolves `FROM nginx:latest` as `localhost/nginx:latest` (treating bare names as local registry), while Docker resolves it as `docker.io/library/nginx:latest`. The build will fail without normalization. Patching in the library avoids forcing every consumer to know about this podman-specific quirk.
