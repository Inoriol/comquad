## 🗺️ Roadmap & Next Steps

### Difficulty: Easy



### Difficulty: Medium

* **`logs` command improvement** *(Medium)* — Pass `--output=short-iso` to journalctl to strip raw systemd metadata (boot ID, machine ID lines). Add `-n/--tail <N>` and `--since <time>` flags. Extend `FollowLogs` to include `.network` and `.volume` unit logs alongside containers.

* **`ps` command improvement** *(Medium)* — Reformat output to match `docker compose ps` style: container name, image, command, status, ports. Run `podman ps --format json` filtered by `com.comquad.project` label, merge with D-Bus active/sub state. Show networks and volumes in a separate section below. Replace the current bare systemd table.

### Difficulty: Hard

* **Lifecycle Integration Testing** *(Hard)* — End-to-end sandbox execution suite. Build a privileged OCI image containing podman, Go, podlet, and systemd. Run `comquad up` / `down` inside it via podman-in-podman. Validate that quadlet files are generated, units start, and state is correctly written and cleaned up. Requires significant CI infrastructure work.

## ✅ Resolved

* **`--dry-run` for `up`** — Runs the full preprocess → transpile → cook pipeline into a temporary preview directory, then prints each generated quadlet file alongside the target path it *would* be written to, plus image build/pull actions that *would* be taken. Nothing is written to the systemd directory, no state is registered, and no units are started. 12 tests added in `dryrun_test.go`.
* **`GetBuildInfo` edge case tests** — 13 tests added: `buildArgValue` with nil, string, empty string, bool true/false, int, int64, float64 whole number, float64 with decimal; `GetBuildInfo` with empty context, empty args map, non-string arg types, and labels alongside build config.
* **Verbosity improvements** — `start`, `stop`, `restart`, and `down` now use `logger.Print` instead of bare `fmt.Printf`, so their operational messages are suppressed by `--quiet`. Added `--quiet`/`-q` persistent root flag that suppresses all non-error output across all commands. `logger.Error` continues to always write to stderr.
* **Orchestrator package tests** — `SystemdClient` and `StateStore` interfaces extracted into `internal/deploy`. Orchestrator uses injected factory functions instead of direct construction calls. 88 tests added across `resolve_test.go`, `lifecycle_test.go`, `down_test.go`, `up_test.go`, and `commands_test.go`.
* **Transpile package tests** — `NewPodletRunner` and `Transpile` covered via a fake `podlet` shell script injected through PATH. 10 tests added.

* **`exec` command broken** — Container name was derived from a full path instead of the filename; fixed with `filepath.Base`.
* **`--dry-run` on `regenerate` non-functional** — Flag was registered but never passed to `Regenerate()`; wired through.
* **`down` networks not deleted** — `RemoveNetworks`/`RemoveVolumes` errors were silently swallowed; now returned and printed to stderr.
* **`checkCmd` created target directory as side effect** — Replaced `os.MkdirAll` with a temp-file write probe.
* **`logger.Error` suppressed in non-verbose mode** — `Error()` now always writes to stderr regardless of verbose setting.
* **Default network / service attachment inconsistency** — Services are only auto-attached to `cq-default` when that network was actually injected, preventing dangling references.
* **Duplicate labels on re-deploy** — `addProjectLabels` now checks for existing labels before inserting.
* **`RegisterProject` dropped existing `Resources`** — Existing resources are preserved when re-registering after `up`.
* **`$EDITOR` not split on whitespace** — `strings.Fields` used so values like `"vim -o"` or `"code --wait"` work correctly.
* **Build arg coercion (`fmt.Sprintf("%v")`)** — Replaced with a proper type-switch (`buildArgValue`) handling nil, bool, int, and float64.
* **State file written non-atomically** — `Save()` now writes to a temp file and renames atomically.
* **Test state leaking into real state file** — Deploy tests now use an isolated temp dir via `XDG_DATA_HOME`.
* **`COMQUAD_PORT_OFFSET` vs `ROOTLESS_PORT_OFFSET` mismatch** — ARCHITECTURE.md corrected to match code and README.
* **`ListProjects` non-deterministic order** — Results sorted alphabetically by project name.
* **`"down"` status dead code in view** — `viewProject` now correctly distinguishes `healthy` / `degraded` / `down`.
* **Multiple D-Bus connections per operation** — `Down()` and `Stop()` now share a single connection across their sub-calls.
* **Compose file read 3× in `Up()`** — `composeData` bytes reused; `preprocess()` no longer reads from disk.
* **`podlet` not validated at startup** — `NewPodletRunner` returns an error immediately if `podlet` is not in PATH.
* **stdin pipe leak in `podlet_runner.go`** — Pipe is closed if `cmd.Start()` fails.
