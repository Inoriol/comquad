## 🗺️ Roadmap & Next Steps

The utility is functional, but the following low-level system integrations are targeted for development:

* ~~**`logs` command improvement** — Get cleaner logs - only for current Invocation ID for running containers and full logs for unit for containers on in other statuses~~ ✅
* **`ps` command improvement** — Clean data-frame formatting to aggregate unit statuses cleanly. Agreggate containers, networks and volumes differently.
* **`down` command improvement** — Don't delete .volume resources unless it's specified with tag
* **`up` command improvement** — Add -f tag to emulate vanilla "docker compose up" behavior, where logs of all project units are being followed in read time from the moment project starts
* **`exec` command** — A command to run a command inside a currently running container.
* **`start`,`stop` and `restart` commands** — A commands to manage containers status
* **Better verbosity** — State to user exactly what changes are made during preparation.
* **Dry run mode support** — To show what changes would have been made without actual deployment.
* **Lifecycle Integration Testing** — End-to-end sandbox execution suites to protect the translation pipeline logic. Build a container with podman, go, podlet. Run it with systemd. Priviliged, so it can run podman-in-podman
* **Self-Healing State Management** — An active reconciliation engine to sync `projects.json` states with active systemd disk states.
