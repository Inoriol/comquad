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

- [ ] **No `--version` flag** on root command
- [ ] **No shell completion generation** — Cobra supports `completion` subcommand for bash/zsh/fish but it's not wired up
- [ ] **No standalone `comquad build` command** — only possible via `comquad up --build`
- [ ] **No config file / global defaults** — no way to set project-level or user-level defaults
- [ ] **No `comquad ls` alias for `comquad list`**
- [ ] **No project-level health status summary** (beyond `view` table)

---

## 🧹 Code Quality & Refactoring

- [x] **`cmd/comquad/main.go` is monolithic (369 lines)** — Fixed: split into per-command files (up.go, down.go, list.go, logs.go, ps.go, check.go, view.go, edit.go, start.go, stop.go, restart.go, regenerate.go, exec.go).
- [x] **`internal/orchestrator/orchestrator.go` is too large (683 lines)** — Split into orchestrator.go (core/Up), down.go (Down/stopUnits), images.go (handleImages/printDryRun), pipeline.go (helpers).
- [x] **`internal/cooker/engine.go` is too large (781 lines)** — Split into engine.go (core), references.go (cross-unit rewriting), ports.go (port offsetting), labels.go (SELinux/project labels, network aliases, systemd optimizations).
- [x] **Mixed output approaches** — Fixed: all output now routed through logger.Print/Printf.
- [x] **`listContainersFromPodman` calls `podman inspect` per-container** — Fixed: batched into single `podman inspect` call via `batchGetExposedPorts`.
- [ ] **`handleImages` re-reads container files from disk** — Files were already written and could be parsed in-memory from the cooker output. Unnecessary disk I/O.
- [x] **Hardcoded string `"unknown"` service name in `orchestrator.go:481`** — Fixed: changed to empty string.
- [x] **Magic numbers everywhere** — Fixed: extracted named constants in dbus.go and orchestrator.go.
- [x] **`execCommand` package-level var in `log.go`** — Fixed: replaced with injectable `newJournalCmd` field on Orchestrator.
- [x] **`renderEntry` has unreachable/confusing branch** — Fixed: added clarifying comment (branch is reachable for entries without systemd unit metadata).
- [ ] **`labelFields` tokenizer in `cooker/engine.go:332-372`** — Hand-written tokenizer for label parsing is complex and error-prone. Could use shell-style parsing library.
- [x] **No godoc on many exported functions** — Fixed: added godoc to MatchFirstContainer, MatchAllContainers, MatchNetworkOrVolume.

---

## 🧪 Testing Gaps

### Missing Unit Tests

- [ ] **`handleImages`** — Complex image-building logic with pull strategies has zero direct test coverage
- [ ] **`stopUnits` / `verifyUnitsStopped`** — Container/network/volume stopping logic
- [ ] **`offsetPorts` in cooker** — Port offset resolution with conflict detection tested only indirectly
- [ ] **`discoverResources` in dbus.go** — Podman JSON output parsing and resource grouping
- [ ] **`removePodmanResources` in dbus.go** — Network/volume removal and error handling
- [ ] **`runJournalctlJSONFollow`** — Complex goroutine + buffered output logic (4 select cases) has zero test coverage
- [ ] **`Regenerate` orchestrator command** — Entire state reconstruction pipeline
- [ ] **`Build()` in build package** — Only tag generation and pull strategy parsing tested
- [ ] **`PullImage()`** — No tests for pull with different strategies
- [ ] **`parseContainer` (podman JSON parsing)** — Only tested through full ps pipeline, not in isolation
- [ ] **`formatTimeAgo`** — Only tested through ps output capture
- [ ] **`StringMap.UnmarshalYAML` / `MarshalYAML`** — No direct tests for edge cases (empty list, mixed types, YAML null)
- [ ] **Main `Execute()` orchestrator pipeline** — No unit test for full preprocess→transpile→cook→deploy flow

### Flaky / Skipped Tests

- [ ] **~10 integration tests use `time.Sleep(2-3s)`** — Race-prone. Should use polling helpers instead of fixed sleeps
- [ ] **3 unit tests require `SKIP_PODLET_TESTS` env var to skip** — Should use `testing.Short()` or a proper build tag
- [ ] **`TestUp_InvalidYamlReturnsError`** accepts either podlet-missing or strategy-invalid error — ambiguous coverage
- [ ] **Multiple log tests collect output but never assert on content** — `result := ...; _ = result` in several tests

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

- [ ] `compose.yaml` with real `build:` blocks
- [ ] `comquad ps` with real output verification
- [ ] `comquad edit` with actual file modifications (only `--no-reload` tested)
- [ ] `comquad exec` with interactive TTY
- [ ] `comquad follow-logs` (`--follow` flag)
- [ ] Concurrent `comquad up` on same project
- [ ] Network isolation (services on different networks)
- [ ] Upgrading a project (deploy, modify compose, redeploy)
- [ ] `down` when systemd units are in `failed` state
- [ ] Behavior with unresponsive podman
- [ ] Large/complex compose files (10+ services)

---

## 📄 Documentation Gaps

- [ ] **No `ROOTLESS_PORT_OFFSET` env var documentation** in README — only briefly mentioned in Architecture doc
- [ ] **No `NO_COLOR` env var documentation**
- [ ] **No example compose files** in the repository — `tests/integration/testdata/` is minimal
- [ ] **No CONTRIBUTING.md** or development setup guide
- [ ] **No man page** or extended help beyond cobra `--help`
- [ ] **CHANGELOG.txt uses plain text format** — Manual `====` headings, not Markdown. Works but prevents nice GitHub rendering
- [ ] **`projects.json` format has no version field** — No forward compatibility guarantee for state file schema evolution

---

## ✨ UX Improvements

- [ ] **No progress indication during `up`** — Pipeline is silent between stages unless `-v` is used. Add at least spinner or stage messages.
- [ ] **No confirmation prompt before `comquad down`** — Destructive action runs immediately without asking
- [ ] **`comquad edit` fallback `vi` may not exist** — Systems with only `vim`/`nano` get a confusing error. Should check PATH.
- [ ] **`comquad view <project>` shows raw file content without context** — No header indicating which file is being shown
- [ ] **No `comquad --help` examples section** — Complex flags like `--since` format deserve usage examples
- [ ] **`comquad ps -a` sorting** — Exited containers mixed into the table could be sorted or grouped for clarity
- [ ] **Error recovery UX** — When `startUnits` fails after files are written, user is told to manually `comquad down` to clean up

---

## 🗺️ Long-Term Roadmap Goals

For reference, these are tracked in [ROADMAP.md](./ROADMAP.md):

- [ ] Bypass engine layer for unsupported compose directives
- [ ] Native secrets management
- [ ] Docker Swarm / Eclipse BlueChi integration
- [ ] Pure systemd-driven build workflow (deprecate host-side build interception)
