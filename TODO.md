## 🗺️ Roadmap & Next Steps

### Uncategorized

* **Review of function names** The singular name is misleading: `MatchContainer` doesn't mean "match exactly one", it means "return one". A caller can't tell if other matches exist. Rename to `MatchFirstContainer` / `MatchAllContainers` or similar. (`internal/orchestrator/resolve.go`). Review all code functions name for clear distinguish

- [x] **Cooking of unit names** cooker phase now rewrites service names in [Unit] section (After=, Requires=, etc.)

* **Exposed ports are not part of PS command** - we need to add exposed ports to ps command to mimir docker compose ps behavior

### Difficulty: Hard

* **Lifecycle Integration Testing** *(Hard)* — End-to-end sandbox execution suite. Build a privileged OCI image containing podman, Go, podlet, and systemd. Run `comquad up` / `down` inside it via podman-in-podman. Validate that quadlet files are generated, units start, and state is correctly written and cleaned up. Requires significant CI infrastructure work.

### Bugs & Robustness

* **`addSELinuxToVolume` missing single-part case** — When Volume= has only one part (e.g. `Volume=appvol` with no colon), `:z` is not appended. Should handle `len(parts) == 1` case. (`internal/cooker/engine.go:547`)
* **`findComposeFile` doesn't verify regular file** — `os.Stat` returns nil for directories too. If a directory is named `compose.yaml`, it would be accepted as a valid compose file. Add `info.Mode().IsRegular()` check. (`internal/orchestrator/orchestrator.go:590`)
* **Port offsetting loop has no upper bound** — `offsetPorts()` uses `for { finalPort++; ... }` with no max port check. Could infinite-loop if all ports in range are claimed. Add check against 65535 and return an error if no available port found. (`internal/cooker/engine.go:327`)
* **`rewriteReferences` non-deterministic order** — Iterates over `renameMap` (Go map) without sorting. If two old names are substrings of each other (e.g. `cq-app` and `cq-application`), replacement order matters and could produce incorrect results. Sort keys before iterating (longest first would be safest). (`internal/cooker/engine.go:144`)
* **`handleImages` and `printDryRun` non-deterministic** — Both iterate over `buildInfo` map without sorting, causing non-deterministic output order between runs. Sort service names before iterating. (`internal/orchestrator/orchestrator.go:360, 454`)
* **SELinux detection data race** — `IsSELinuxEnabled()` and `SELinuxMode()` use package-level vars (`selinuxEnabled`, `selinuxMode`) without synchronization. Concurrent calls during initialization could cause a data race. Add `sync.Once` or mutex. (`internal/preprocess/selinux.go:40`)

### Code Smells

* **Remove `containerFileToUnitName` no-op alias** — Unexported `containerFileToUnitName` is just `return ContainerFileToUnitName(filePath)`. Remove the alias and use the exported version directly at all call sites. (`internal/orchestrator/orchestrator.go:575`)
* **Remove unused `Engine.ForceBuild` field** — Field on `build.Engine` struct is never read anywhere in the codebase. (`internal/build/engine.go:26`)
* **Remove dead-code stub `Engine.HandleBuild`** — Returns `nil` unimplemented. The orchestrator calls `BuildService()` directly instead. Either implement or remove. (`internal/build/engine.go:116`)
* **`splitCombinedLabels` missing quote handling** — Uses `strings.Fields()` which splits on all whitespace. If a label value contains spaces (even quoted), it would split incorrectly. Consider proper tokenizer. (`internal/cooker/engine.go:216`)

### Test Gaps

* **No tests for `regenerate.go`** — `Regenerate()` orchestrator method has zero test coverage.
* **No tests for `edit.go`** — `Edit()` orchestrator method has zero test coverage.
* **No tests for `ps.go`** — `Ps()` orchestrator method has zero test coverage.

### Quick Wins

* **Extract `ensureProjectDeployed()` helper** — 7+ methods (`View`, `Edit`, `Exec`, `Logs`, `Ps`, `FollowLogs`, `Regenerate`) repeat the pattern: create state manager → `GetProject` → fail if not exists. Extract into a single helper.
* **Add `--dry-run` to `down`, `start`, `stop`, `restart`** — `regenerate` already supports it. Adding dry-run to lifecycle commands would improve safety for users.
