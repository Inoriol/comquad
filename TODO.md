## 🗺️ Roadmap & Next Steps

The utility is functional, but the following low-level system integrations are targeted for development:

* **`logs` command improvement** — Get cleaner logs - only for current Invocation ID for running containers and full logs for unit for containers on in other statuses
* **`ps` command improvement** — Clean data-frame formatting to aggregate unit statuses cleanly. Agreggate containers, networks and volumes differently
* **`view` command** — To view systemd units for service or full specific project
* **`edit` workflow** — A command pipeline to open active quadlets inside an `$EDITOR` buffer and instantly reload systemd.
* **`exec` command** — A command to run a command inside a currently running container.
* **Better verbosity** — State to user exactly what changes are made during preparation.
* **Dry run mode support** — To show what changes would have been made without actual deployment.
* **Lifecycle Integration Testing** — End-to-end sandbox execution suites to protect the translation pipeline logic.
* **Self-Healing State Management** — An active reconciliation engine to sync `projects.json` states with active systemd disk states.
