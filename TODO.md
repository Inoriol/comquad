## 🗺️ Roadmap & Next Steps

The utility is functional, but the following low-level system integrations are targeted for development:

* **`logs` command improvement** — Get cleaner logs - more options and include logs for volumes and networks. Don't show user and other systemd info left on logs by default
* **`ps` command improvement** — Clean data-frame formatting to aggregate unit statuses cleanly. Aggregate containers, networks and volumes differently. Get status, running time, cpu and memory from systemd. Get everything else from podman inspect or podman ps --format json
* **Lifecycle Integration Testing** — End-to-end sandbox execution suites to protect the translation pipeline logic. Build a container with podman, go, podlet. Run it with systemd. Privileged, so it can run podman-in-podman
* **Orchestrator package tests** — Extract dependencies behind interfaces (Filesystem, SystemdClient, CommandRunner) for mocking. Test Up/Down error paths, resolveUnits, collectProjectFiles, and lifecycle scenarios.
* **GetBuildInfo edge case tests** — Test empty string context, build with labels, empty args map. The coercion bug (bool/int/nil arg values) is fixed; typed coverage is still missing.
* **Verbosity improvements** — Add verbose logging to `down`, `start`, `stop`, `restart` commands (currently use plain `fmt.Printf`). Consider adding `--quiet` flag to suppress non-error output.

## ✅ Resolved

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
