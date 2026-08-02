## 🗺️ Roadmap & Next Steps

For long term goals refer to [Roadmap](./ROADMAP.md).

---

## 🔴 Release Blockers (must fix before v0.1.0)

These issues directly break core functionality or the release process:

- [ ] **Bug: `handleImages` skips all tagged images, not just build-generated ones** — `orchestrator.go` checks `strings.Contains(image, ":")` which skips both build-tagged images AND user-specified tagged images (e.g. `postgres:15`, `nginx:alpine`). This means the most common compose image pattern never gets pulled or validated. The check should match only the build pattern (`<projectName>-<service>:latest`).
- [ ] **Bug: `ImageExists` silently swallows all errors** — `build/engine.go` returns `false` for any `podman image inspect` error instead of only exit code 125. A broken podman install is indistinguishable from "image missing", causing silent build skips.
- [ ] **Bug: `--verbose` flag only works with `up` command** — `SetVerbose` is only called from `upCmd`. Commands like `down`, `ps`, `start`, `restart`, `logs` cannot enable verbose output.
- [ ] **Bug: `--quiet` flag is bypassed by direct `fmt.Println` usage** — `view`, `ps`, `list`, `regenerate`, and dry-run previews all use `fmt.Println` directly instead of the logger, so `--quiet` cannot suppress their output.
- [ ] **`.goreleaser` has no extension** — goreleaser looks for `.goreleaser.yaml` or `.goreleaser.yml` by default. Rename to `.goreleaser.yml` or add `--config` flag to build scripts.

---

## 🐛 Bugs

- [ ] **`MatchNetworkOrVolume` double-strips extensions** — `resolve.go:79-81` strips `.network` then strips `.volume` from the already-stripped string. Works by accident since only one extension is present, but is logically incorrect and fragile.
- [ ] **`exec` podman PATH check is after building the command** — `exec.go` calls `exec.Command("podman", ...)` and logs before checking `LookPath("podman")`. User gets a confusing intermediate message if podman is missing.
- [ ] **`StringMap` type mismatch for volume labels in `preprocess/engine.go`** — The map value type is `interface{}` but the case checks `StringMap` which never matches. Volume label injection falls through to the `default` case (works, but list-format label preservation logic is dead code).
- [ ] **`logger.SetQuiet` does not suppress `fmt.Println` output** — See release blocker above. All direct `fmt.Println` calls should be routed through the logger.

---

## 🛡️ Security

- [ ] **No project name sanitization** — Project names derived from directory names pass directly into file paths and systemd unit names. A directory named with special characters (`../../`, spaces) could cause issues.
- [ ] **State file permissions** — `projects.json` at `~/.local/share/comquad/` is written with default permissions (0644). Should ensure restricted permissions on sensitive data.
- [ ] **No validation of external podman JSON output** — `ps_podman.go` and `dbus.go` trust podman's JSON output with only basic type assertions. Malformed output could cause panics.
- [ ] **User-controlled strings flow into `exec.Command`** — Project names, service names, and command arguments all flow into `exec.Command`. While Go's exec model prevents shell injection, arguments with special characters could behave unexpectedly.

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

- [ ] **`cmd/comquad/main.go` is monolithic (369 lines)** — All commands in one file. Should split into per-command files (e.g. `commands/up.go`, `commands/down.go`, `commands/logs.go`).
- [ ] **`internal/orchestrator/orchestrator.go` is too large (683 lines)** — Combine `Up`, `Down`, `handleImages`, `stopUnits`, `verifyUnitsStopped`, `printDryRun`. Should be split into focused files.
- [ ] **`internal/cooker/engine.go` is too large (781 lines)** — Monolithic cooker handling rename, reference rewrite, SELinux, ports, labels, network aliases, and systemd optimizations.
- [ ] **Mixed output approaches** — Some commands use `logger.Print/Action/Success`, others use `fmt.Println` directly. Standardize on the logger.
- [ ] **`listContainersFromPodman` calls `podman inspect` per-container** — For N containers this makes N separate podman calls. Could batch into one `podman inspect <c1> <c2> ...`.
- [ ] **`handleImages` re-reads container files from disk** — Files were already written and could be parsed in-memory from the cooker output. Unnecessary disk I/O.
- [ ] **Hardcoded string `"unknown"` service name in `orchestrator.go:481`** — Meaningless service name in log output for non-build images.
- [ ] **Magic numbers everywhere** — `15*time.Second` WaitForUnit timeout, `10*time.Second` startUnits wait, `30*time.Second` D-Bus context, `500*time.Millisecond` poll/flush intervals. None are configurable.
- [ ] **`execCommand` package-level var in `log.go`** — Testing hook should use a real interface, not a mutable global.
- [ ] **`renderEntry` has unreachable/confusing branch** — `log.go` sets unit to `"?"` when empty but the logic around it is unclear.
- [ ] **`labelFields` tokenizer in `cooker/engine.go:332-372`** — Hand-written tokenizer for label parsing is complex and error-prone. Could use shell-style parsing library.
- [ ] **No godoc on many exported functions** — `ContainerFileToUnitName`, `NetworkFileToUnitName`, `VolumeFileToUnitName`, `MatchFirstContainer`, `MatchAllContainers`, `MatchNetworkOrVolume` lack godoc comments.

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
