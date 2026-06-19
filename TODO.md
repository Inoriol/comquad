## 🗺️ Roadmap & Next Steps

The utility is functional, but the following low-level system integrations are targeted for development:

* **`down` fix** — Seems networks are not getting properly deleted. Run podman network rm $(podman network ls --filter "label=com.comquad.project=projectname" -q) after deleting actual .network file and reloading
* **`logs` command improvement** — Get cleaner logs - more options and include logs for volumes and networks. Don't show user and other systemd info left on logs by default
* **`ps` command improvement** — Clean data-frame formatting to aggregate unit statuses cleanly. Agreggate containers, networks and volumes differently. Get status, running time, cpu and memory from systemd. Get everything else from podman inspect or podman ps --format json
* **Dry run mode support** — To show what changes would have been made without actual deployment.
* **Lifecycle Integration Testing** — End-to-end sandbox execution suites to protect the translation pipeline logic. Build a container with podman, go, podlet. Run it with systemd. Priviliged, so it can run podman-in-podman
* **Orchestrator package tests** — Extract dependencies behind interfaces (Filesystem, SystemdClient, CommandRunner) for mocking. Test Up/Down error paths, resolveUnits, collectProjectFiles, and lifecycle scenarios.
* **GetBuildInfo edge case tests** — Test empty string context, build with labels, empty args map, and complex env var types (bool, number, empty string)
* **Verbosity improvements** — Add verbose logging to `down`, `start`, `stop`, `restart` commands. Consider adding `--quiet` flag to suppress non-error output.
