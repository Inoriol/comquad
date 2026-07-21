## 🗺️ Roadmap & Next Steps

This file contains short term goals that can be achieved within short timeframe. For long term goals reffer to [Roadmap](./ROADMAP.md) guide.

### Uncategorized

* **Review of function names** The singular name is misleading: `MatchContainer` doesn't mean "match exactly one", it means "return one". A caller can't tell if other matches exist. Rename to `MatchFirstContainer` / `MatchAllContainers` or similar. (`internal/orchestrator/resolve.go`). Review all code functions name for clear distinguish

* **Cleanup of "myapp" that pops up during test** - after go testing, container and network of "myapp" keep persisting in system

* **Better check** - let's make sure that podman 4.4 (version where quadlets appeared) required for running (comquad check)

* **Verbosity improvements** - some key stuff from verbosity needs to be done by default (like port remapping)

### Difficulty: Hard

* **Lifecycle Integration Testing** *(Hard)* — End-to-end sandbox execution suite. Build a privileged OCI image containing podman, Go, podlet, and systemd. Run `comquad up` / `down` inside it via podman-in-podman. Validate that quadlet files are generated, units start, and state is correctly written and cleaned up. Requires significant CI infrastructure work.

### Bugs & Robustness

* **`addSELinuxToVolume` missing single-part case** — ~~FIXED~~ When Volume= has only one part (e.g. `Volume=appvol` with no colon), `:z` was not appended. Now handles `len(parts) == 1`. Also replaced `strings.Contains` z/Z check with proper comma-delimited token matching to avoid false positives like `zoo`. (`internal/cooker/engine.go`)

* **`findComposeFile` doesn't verify regular file** — ~~FIXED~~ `os.Stat` returns nil for directories too. Added `info.Mode().IsRegular()` check. (`internal/orchestrator/orchestrator.go`)

* **Port offsetting loop has no upper bound** — ~~FIXED~~ `offsetPorts()` now checks `finalPort > 65535` and returns an error if no available port found. (`internal/cooker/engine.go`)

* **`rewriteReferences` non-deterministic order** — ~~FIXED~~ Keys are now sorted longest-first before iterating to avoid partial prefix match issues. (`internal/cooker/engine.go`)

* **`handleImages` and `printDryRun` non-deterministic** — ~~FIXED~~ Both now sort service names before iterating `buildInfo`. (`internal/orchestrator/orchestrator.go`)

* **SELinux detection data race** — `IsSELinuxEnabled()` and `SELinuxMode()` use package-level vars (`selinuxEnabled`, `selinuxMode`) without synchronization. Concurrent calls during initialization could cause a data race. Add `sync.Once` or mutex. (`internal/preprocess/selinux.go`)

* **`runJournalctlJSONFollow` goroutine leak on error** — If `cmd.Wait()` returns an error in the main goroutine, the scanner goroutine blocks on `scanner.Scan()` forever because `stdout` is never closed. Use `cmd.Wait()` in a separate goroutine or close stdout/stderr properly on error. (`internal/orchestrator/log.go:332-347`)

* **`collectJournalEntries` not checking `scanner.Err()`** — Scanner errors (e.g., broken pipe) are silently swallowed. Should check `scanner.Err()` after the loop. (`internal/orchestrator/log.go:274-283`)

* **`splitCombinedLabels` missing quote handling** — Uses `strings.Fields()` which splits on all whitespace. If a label value contains spaces (even quoted), it would split incorrectly. Consider proper tokenizer. (`internal/cooker/engine.go`)

* **`discoverResources` container parsing is fragile** — Uses `|` as delimiter in `podman ps` output format. If a container name contains `|`, parsing breaks silently. No validation that project name is non-empty after parsing. (`internal/deploy/dbus.go:346-428`)

* **`removePodmanResources` silently swallows partial failures** — When removing multiple resources, some may fail and others succeed. Individual failures are not surfaced clearly, risking orphaned resources. (`internal/deploy/dbus.go:208-250`)

* **`offsetPorts` re-reads files after modifying** — File is re-read just to update a single line even though content was already read in the first pass. Could be cached. (`internal/cooker/engine.go:419-435`)

* **`addSELinuxToVolume` z/Z check uses substring** — `strings.Contains(parts[2], "z")` matches false positives like `zoo`. Replaced with comma-delimited token check. (`internal/cooker/engine.go`)

### Code Smells

* **Remove `containerFileToUnitName` no-op alias** — ~~FIXED~~ Removed the alias and replaced all 5 call sites with `ContainerFileToUnitName` directly. (`internal/orchestrator/orchestrator.go`, `lifecycle.go`, `edit.go`)

* **Remove unused `Engine.ForceBuild` field** — ~~FIXED~~ Field on `build.Engine` struct was never read anywhere in the codebase. Removed. (`internal/build/engine.go:26`)

* **Remove dead-code stub `Engine.HandleBuild`** — ~~FIXED~~ Returns `nil` unimplemented. The orchestrator calls `BuildService()` directly instead. Removed. (`internal/build/engine.go:116`)

* ~~**`normalizeImage` incorrectly classifies registry names**~~ — **DONE**: Added `isRegistryWithPort()` helper that checks if `:` is followed by digits (port number) vs alphanumeric (image tag). Distinguishes `localhost:5000` (registry) from `myapp:v1` (image with tag). (`internal/preprocess/engine.go`)

* **`addProjectLabels` uses string comparison for label detection** — Exact string match `trimmed == "Label=com.comquad.project="+c.ProjectName` is fragile. If label value contains trailing whitespace or different quoting, the check fails and a duplicate label is injected on re-deploy. (`internal/cooker/engine.go:557-563`)

* **`logs` / `FollowLogs` `--since` flag without validation** — The `since` string is passed directly to `journalctl` without validation. An invalid time format causes `journalctl` to fail at runtime with a confusing error. Should validate or catch the error. (`internal/orchestrator/log.go:251`, `internal/orchestrator/log.go:403`)

* **`ps` command uses `ListAllUnits()` for project filtering** — `ListAllUnits()` fetches ALL systemd units on the system and filters in Go. Inefficient compared to using a filtered query. Called on every `ps` invocation. (`internal/orchestrator/ps.go:38-56`)

* **`exec` command does not validate `podman` exists** — Unlike `up` which checks for `podlet` upfront, the `exec` command does not verify `podman` is in PATH. (`internal/orchestrator/exec.go:73`)

* **`edit` command does not validate `$EDITOR` exists** — The editor binary is not validated to exist before launching. (`internal/orchestrator/edit.go:83-90`)

* **`StateStore` interface does not prevent concurrent map access** — The `Projects` map is shared without mutex protection. If two goroutines call `RegisterProject` concurrently, map access is unsynchronized. (`internal/deploy/interfaces.go`)

* **`StateStore` interface missing `Delete` / `Remove` method** — The interface has `RegisterProject` and `UnregisterProject`, but `UnregisterProject` is only called from `Down()`. Minor design concern. (`internal/deploy/interfaces.go`)

* **`verifyUnitsStopped` swallows D-Bus errors** — If D-Bus returns an error for a unit, the unit is silently skipped. If all units return errors, `activeUnits` stays empty and the function returns `nil` (success), falsely indicating all units are stopped. (`internal/orchestrator/orchestrator.go:568-589`)

* **`editProject` auto-restarts without verification** — Failed unit restarts are only printed to stdout. The unit remains in a potentially inconsistent state. Should return an error or log to stderr. (`internal/orchestrator/edit.go:143-154`)

* **`discoverResources` swallows Podman errors** — If `podman ps -a` fails (e.g., daemon not running), the function returns an empty slice silently. The `regenerate` command will produce an empty state without warning the user. (`internal/deploy/dbus.go:361-365`)

* **`splitCombinedLabels` drops empty label values** — If a `Label=` line has no value, `strings.Fields` returns an empty slice and nothing is appended. The label is silently dropped. (`internal/cooker/engine.go`)

### Quick Wins

* ~~**Extract `ensureProjectDeployed()` helper**~~ — **DONE**: Extracted `ensureProjectDeployed()` helper in `internal/orchestrator/orchestrator.go`. Updated 8 methods (`Ps`, `Down`, `View`, `Edit`, `Exec`, `Logs`, `FollowLogs`, `resolveUnits`) to use it. Returns `(StateStore, ProjectState, error)` to support both state access and operations like `UnregisterProject`.

* **Add `--dry-run` to `down`, `start`, `stop`, `restart`** — `regenerate` already supports it. Adding dry-run to lifecycle commands would improve safety for users.

* ~~**`printPsTable` column widths have hardcoded minimums**~~ — **DONE**: Column minimums now use `max(len("HEADER"), minimum)` to ensure headers are never truncated even if minimums are reduced in the future. (`internal/orchestrator/ps.go`)

* **`ps` command `formatTimeAgo` can produce negative times** — If the container was created in the future (clock skew), `time.Since(t)` is negative and returns "recently". Minor UX issue. (`internal/orchestrator/ps.go:151-185`)

* ~~**`logger.Error` does not respect dynamic `noColor`**~~ — **DONE**: `colorize()` now checks `NO_COLOR` environment variable dynamically on each call, in addition to the `noColor` variable set at init. (`internal/logger/logger.go`)

### Integration Test Fixes

* ~~**Tests rely on compose `name:` matching directory name**~~ — **FIXED**: Updated `WriteCompose` to return project name, added `--name <project>` to all `up` calls. Tests no longer depend on `t.TempDir()` naming conventions.

* ~~**SELinux absence test missing `down` cleanup**~~ — **FIXED**: Added `t.Cleanup` with `down` call for uniformity. Note: `SELinuxPresent(t)` skip condition is correct — it detects SELinux mount availability, not enforcement mode. Comquad's `:z` injection triggers on presence detection via `/sys/fs/selinux/enforce` file content.

* ~~**`TestExec_AmbiguousService_Errors` tests wrong scenario**~~ — **FIXED**: Renamed to `TestExec_NonexistentService_Errors` to accurately describe the tested scenario.

* **Remaining**: Consider adding a test for true ambiguous service matching (two services with overlapping names). Currently untested.
