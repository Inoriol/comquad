## 🗺️ Roadmap & Next Steps

The utility is functional, but the following low-level system integrations are targeted for development:

* **`logs` command improvement** — Get cleaner logs - more options and include logs for volumes and networks
* **`ps` command improvement** — Clean data-frame formatting to aggregate unit statuses cleanly. Agreggate containers, networks and volumes differently.
* **`down` command improvement** — ~~Don't delete .volume resources unless it's specified with tag.~~ Networks and volumes are now removed. Networks are always removed; volumes require `-v` flag.
* **Better verbosity** — State to user exactly what changes are made during preparation.
* **Dry run mode support** — To show what changes would have been made without actual deployment.
* **Lifecycle Integration Testing** — End-to-end sandbox execution suites to protect the translation pipeline logic. Build a container with podman, go, podlet. Run it with systemd. Priviliged, so it can run podman-in-podman
* **Self-Healing State Management** —Add "regenerate" command to restore state from labels. Also auto-run it in case state has not been found during executions of something
* **Orchestrator package tests** — Extract dependencies behind interfaces (Filesystem, SystemdClient, CommandRunner) for mocking. Test Up/Down error paths, resolveUnits, collectProjectFiles, and lifecycle scenarios.
* **GetBuildInfo edge case tests** — Test empty string context, build with labels, empty args map, and complex env var types (bool, number, empty string)
