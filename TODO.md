## 🗺️ Roadmap & Next Steps

For long term goals refer to [Roadmap](./ROADMAP.md).

---
## 🧩 Missing Features

- [x] **No `--version` flag** on root command — Added: `version` variable set via ldflags, displayed in rootCmd.Version.
- [x] **No shell completion generation** — Already wired up by default in Cobra v1.8.0. Works for bash/zsh/fish/powershell via `comquad completion`.
- [ ] **No standalone `comquad build` command** — `build:` blocks are explicitly rejected for now. Infrastructure for `.image`/`.build` quadlet file handling is in place (cooker, resolve, lifecycle, down, regenerate, logs, edit, view).
- [ ] **No config file / global defaults** — no way to set project-level or user-level defaults
- [x] **No `comquad ls` alias for `comquad list`** — Added: `Aliases: []string{"ls"}` on listCmd.
- [x] **`.image` and `.build` quadlet file types** — Added: full support across cooker, lifecycle, resolve, down, view, edit, logs, and regenerate.
- [ ] **No project-level health status summary** (beyond `view` table)

---
---

## 🧪 Testing Gaps

### Missing Unit Tests

- [ ] **`handleImages`** — Complex image-building logic with pull strategies has zero direct test coverage
- [x] **`stopUnits` / `verifyUnitsStopped`** — Added: 5 direct unit tests for stopUnits with mock D-Bus (stopsOnlyContainers, noContainerFiles, multipleContainers, propagatesError, emptyProjectFiles). verifyUnitsStopped already had direct tests.
- [ ] **`offsetPorts` in cooker** — Port offset resolution with conflict detection tested only indirectly
- [ ] **`discoverResources` in dbus.go** — Podman JSON output parsing and resource grouping
- [ ] **`removePodmanResources` in dbus.go** — Network/volume removal and error handling
- [ ] **`runJournalctlJSONFollow`** — Complex goroutine + buffered output logic (4 select cases) has zero test coverage
- [ ] **`Regenerate` orchestrator command** — Entire state reconstruction pipeline
- [ ] **`Build()` in build package** — Only tag generation and pull strategy parsing tested
- [ ] **`PullImage()`** — No tests for pull with different strategies
- [x] **`parseContainer` (podman JSON parsing)** — Added: 6 direct unit tests (basicFields, serviceNameDerivation, exitedContainer, nilInput, emptyName, noNamesField). Also 3 parsePorts + 4 parseStringSlice tests.
- [x] **`formatTimeAgo`** — Added: 10 direct unit tests covering now, seconds, minutes, hours, days, 1+ weeks, future dates. Also 3 formatPorts, 3 formatCreated, 4 truncate tests.
- [x] **`StringMap.UnmarshalYAML` / `MarshalYAML`** — Added: 8 direct unit tests (listFormat, mapFormat, emptyList, emptyMap, yamlNull, invalidFormat, marshalYAML, marshalYAML_empty).
- [ ] **Main `Execute()` orchestrator pipeline** — No unit test for full preprocess→transpile→cook→deploy flow

### Test Infrastructure

- [x] **No CI pipeline** — Added: `.github/workflows/test.yml` with build, vet, short, race, and coverage steps
- [x] **Integration tests build binary inside container** — Fixed: binary is now pre-built on host and mounted into container
- [x] **Integration Containerfile uses `fedora:43`** — Fixed: changed to `fedora:41` (stable)
- [x] **`loginctl enable-linger` may silently fail** — Fixed: replaced with direct file creation in `/var/lib/systemd/linger/`
- [x] **No `go test -short` support** — Added: `make test-short` target
- [x] **No `go test -race` or `go test -cover` in Makefile** — Added: `make test-race`, `make test-cover` targets
- [x] **`captureStdout` in unit tests** — Fixed: added `sync.Mutex` serialization for goroutine safety
- [ ] **No fuzzing tests** for YAML or quadlet parsers
- [ ] **No benchmark tests** for any performance-sensitive paths

### Missing Integration Test Scenarios

- [ ] `compose.yaml` with real `build:` blocks — ~~Added: TestUpDown_WithBuildBlocks in build_test.go~~ (test no longer applicable; build blocks are now rejected by preprocessor)
- [x] `comquad ps` with real output verification — Added: TestPs_OutputFormat and TestPs_AllIncludesExitedContainers in ps_integration_test.go
- [x] `comquad edit` with actual file modifications — Added: TestEdit_WithFileModifications in edit_modify_test.go (uses sed as EDITOR)
- [ ] `comquad exec` with interactive TTY
- [ ] `comquad follow-logs` (`--follow` flag)
- [ ] Concurrent `comquad up` on same project
- [x] Network isolation (services on different networks) — Added: TestNetworkIsolation_DifferentNetworks in network_test.go
- [ ] Upgrading a project (deploy, modify compose, redeploy)
- [x] `down` when systemd units are in `failed` state — Added: TestDown_WhenUnitsAreFailed in up_down_test.go
- [ ] Behavior with unresponsive podman
- [ ] Large/complex compose files (10+ services)

---

## 📄 Documentation Gaps

- [x] **No `ROOTLESS_PORT_OFFSET` env var documentation** — Added to README.
- [x] **No `NO_COLOR` env var documentation** — Added to README.
- [ ] **No example compose files** in the repository — `tests/integration/testdata/` is minimal
- [ ] **No CONTRIBUTING.md** or development setup guide
- [ ] **No man page** or extended help beyond cobra `--help`
- [ ] **`projects.json` format has no version field** — No forward compatibility guarantee for state file schema evolution

---

## 🗺️ Long-Term Roadmap Goals

For reference, these are tracked in [ROADMAP.md](./ROADMAP.md):
