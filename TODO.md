## 🗺️ Roadmap & Next Steps

For long term goals refer to [Roadmap](./ROADMAP.md).

---

## 🔴 Release Blockers (must fix before v0.1.0)

These issues directly break core functionality or the release process:

- [x] **Bug: `handleImages` skips all tagged images, not just build-generated ones** — Fixed: `isBuildGeneratedImage()` checks against buildInfo services.
- [x] **Bug: `ImageExists` silently swallows all errors** — Fixed: warns on non-125 exit codes and still returns false.
- [x] **Bug: `--verbose` flag only works with `up` command** — Fixed: moved to rootCmd.PersistentFlags with SetVerbose in PersistentPreRun.
- [x] **Bug: `--quiet` flag is bypassed by direct `fmt.Println` usage** — Fixed: added logger.Printf, routed all output through logger.
- [x] **`.goreleaser` has no extension** — Fixed: renamed to `.goreleaser.yml`.

---

## 🐛 Bugs

- [x] **`MatchNetworkOrVolume` double-strips extensions** — Fixed: strips only the relevant extension.
- [x] **`exec` podman PATH check is after building the command** — Fixed: moved LookPath before exec.Command.
- [x] **`StringMap` type mismatch for volume labels in `preprocess/engine.go`** — Fixed: added `map[string]interface{}` case that preserves existing labels.
- [x] **`logger.SetQuiet` does not suppress `fmt.Println` output** — Fixed (see release blocker above).

---

## 🛡️ Security

- [x] **No project name sanitization** — Fixed: `validateProjectName()` enforces `[a-zA-Z0-9_-]` only.
- [x] **State file permissions** — Fixed: dir created with 0700, temp file chmod 0600 before atomic rename.
- [x] **No validation of external podman JSON output** — Fixed: added nil-safety checks in `parseContainer`.
- [x] **User-controlled strings flow into `exec.Command`** — Addressed by project name validation preventing unsafe chars.

---

## 🧩 Missing Features

- [x] **No `--version` flag** on root command — Added: `version` variable set via ldflags, displayed in rootCmd.Version.
- [x] **No shell completion generation** — Already wired up by default in Cobra v1.8.0. Works for bash/zsh/fish/powershell via `comquad completion`.
- [ ] **No standalone `comquad build` command** — only possible via `comquad up --build`
- [ ] **No config file / global defaults** — no way to set project-level or user-level defaults
- [x] **No `comquad ls` alias for `comquad list`** — Added: `Aliases: []string{"ls"}` on listCmd.
- [ ] **No project-level health status summary** (beyond `view` table)

---

## 🧹 Code Quality & Refactoring

- [x] **`cmd/comquad/main.go` is monolithic (369 lines)** — Fixed: split into per-command files (up.go, down.go, list.go, logs.go, ps.go, check.go, view.go, edit.go, start.go, stop.go, restart.go, regenerate.go, exec.go).
- [x] **`internal/orchestrator/orchestrator.go` is too large (683 lines)** — Split into orchestrator.go (core/Up), down.go (Down/stopUnits), images.go (handleImages/printDryRun), pipeline.go (helpers).
- [x] **`internal/cooker/engine.go` is too large (781 lines)** — Split into engine.go (core), references.go (cross-unit rewriting), ports.go (port offsetting), labels.go (SELinux/project labels, network aliases, systemd optimizations).
- [x] **Mixed output approaches** — Fixed: all output now routed through logger.Print/Printf.
- [x] **`listContainersFromPodman` calls `podman inspect` per-container** — Fixed: batched into single `podman inspect` call via `batchGetExposedPorts`.
- [x] **`handleImages` re-reads container files from disk** — Fixed: Cook() returns CookResult with in-memory FileContents, handleImages/printDryRun accept content map instead of reading from disk.
- [x] **Hardcoded string `"unknown"` service name in `orchestrator.go:481`** — Fixed: changed to empty string.
- [x] **Magic numbers everywhere** — Fixed: extracted named constants in dbus.go and orchestrator.go.
- [x] **`execCommand` package-level var in `log.go`** — Fixed: replaced with injectable `newJournalCmd` field on Orchestrator.
- [x] **`renderEntry` has unreachable/confusing branch** — Fixed: added clarifying comment (branch is reachable for entries without systemd unit metadata).
- [x] **`labelFields` tokenizer in `cooker/engine.go:332-372`** — Rewritten: cleaner string-based implementation using strings.IndexByte. 13 unit tests added.
- [x] **No godoc on many exported functions** — Fixed: added godoc to MatchFirstContainer, MatchAllContainers, MatchNetworkOrVolume.

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

### Flaky / Skipped Tests

- [x] **~10 integration tests use `time.Sleep(2-3s)`** — Fixed: replaced with polling helpers. Added WaitForLogs (polls every 500ms up to 30s). Removed redundant sleeps where AssertUnitActive already polls.
- [x] **3 unit tests require `SKIP_PODLET_TESTS` env var to skip** — Fixed: replaced with `testing.Short()` (standard Go convention).
- [x] **`TestUp_InvalidYamlReturnsError`** — Fixed: now validates error message contains yaml/preprocess/unmarshal keywords.
- [x] **Multiple log tests collect output but never assert on content** — Fixed: TestLogs_StoppedUnit and TestLogs_SpecificService now assert non-empty output. TestFlushEntries_SortsByTimestamp now captures stdout and asserts sort order. TestUp_StateRegistrationError and TestUp_InvalidPullStrategyReturnsError now have proper assertions.

### Test Infrastructure

- [ ] **No CI pipeline** — No `.github/workflows/` or any CI config. Makefile targets exist but nothing runs automatically
- [ ] **Integration tests build binary inside container** — Adds ~30s per run and doesn't test the actual release binary
- [ ] **Integration Containerfile uses `fedora:43`** — Bleeding-edge; should use a stable version
- [ ] **`loginctl enable-linger` may silently fail** — `|| true` in Containerfile masks errors; rootless tests could pass incorrectly
- [ ] **No `go test -short` support** — Can't run only fast tests
- [ ] **No `go test -race` or `go test -cover` in Makefile** — No concurrency safety or coverage measurement
- [ ] **`captureStdout` in unit tests** — Replaces `os.Stdout`, not goroutine-safe, blocks `t.Parallel()`
- [ ] **No fuzzing tests** for YAML or quadlet parsers
- [ ] **No benchmark tests** for any performance-sensitive paths

### Missing Integration Test Scenarios

- [x] `compose.yaml` with real `build:` blocks — Added: TestUpDown_WithBuildBlocks in build_test.go
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

## ✨ UX Improvements

- [x] **No progress indication during `up`** — Fixed: added logger.Action() calls at each pipeline stage (Reading, Preprocessing, Transpiling, Generating quadlet, Handling images, Starting services).
- [x] **No confirmation prompt before `comquad down`** — Fixed: prompts "Are you sure?" when stdin is a terminal. Added `-y`/`--yes` flag to skip. Dry-run also skips prompt.
- [x] **`comquad edit` fallback `vi` may not exist** — Fixed: findDefaultEditor() tries editor, nano, vim, vi in order.
- [x] **`comquad view <project>` shows raw file content without context** — Fixed: printFile() now shows `── <filename> ──` header.
- [x] **No `comquad --help` examples section** — Fixed: added Example fields to up, down, logs, exec, ps, regenerate, edit commands.
- [x] **`comquad ps -a` sorting** — Fixed: running containers first (by name), then exited (by name), then other states.
- [x] **Error recovery UX** — Fixed: cleanup() now called on startUnits failure, removing files and unregistering project.

---

## 🗺️ Long-Term Roadmap Goals

For reference, these are tracked in [ROADMAP.md](./ROADMAP.md):

- [ ] Bypass engine layer for unsupported compose directives
- [ ] Native secrets management
- [ ] Docker Swarm / Eclipse BlueChi integration
- [ ] Pure systemd-driven build workflow (deprecate host-side build interception)
