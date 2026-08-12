## 🗺️ Roadmap & Next Steps

For long term goals refer to [Roadmap](./ROADMAP.md).

### Pre-Push Checklist (v0.3.2)

- [x] **Integration tests** — Updated integration tests after compose2quadlet rewrite:
  - `up_down_test.go` — verified full up/down lifecycle with new quadlet output format
  - `dry_run_test.go` — verified dry-run matches new output (container names, network aliases, exec quoting, section spacing, patched Dockerfile display)
  - `lifecycle_test.go` — verified start/stop/restart with new unit naming
  - `logs_test.go` — verified log streaming still works
  - `exec_test.go` — verified podman exec with new container names
  - `rootless_test.go` — verified port offset logging (≤ 1024 threshold) and rootless paths
  - `selinux_test.go` — updated for relabel=shared on Mount= directives
  - `view_edit_test.go` — updated for new status/name display format
- [x] **`.image` timeout on older podman** — Verified graceful fallback when quadlet generator doesn't produce `.image` service units (podman < 5.5 where `Retry=`/`RetryDelay=` are unrecognized). Warnings logged, no cascade failures.
- [x] **`comquad regenerate`** — Verified state reconstruction from Podman labels works with the updated state file format (files list, resources mapping with new naming including images and builds).
- [x] **Build flow** — Tested `build:` blocks end-to-end with `.build` quadlet generation, Dockerfile FROM normalization, `ImageTag=` defaulting, and healthcheck `ExecStartPre` container references. Builds deploy and containers start correctly.

### Future Security Improvements

- **Tmpfs-backed secrets via `LoadCredential`** — Currently secrets are bind-mounted directly from managed files on disk. Consider generating a companion `.service` service unit file (not a `.container` quadlet) that uses systemd `LoadCredential=` in `[Service]` combined with `Volume=%d/<name>` to mount secrets from systemd's RAM-backed credential directories (`/run/credentials/`). This would keep secret values in tmpfs memory rather than on persistent storage. Initial implementation attempted this using quadlet's `[Service]` pass-through, but `LoadCredential=` + `Volume=` with credential paths didn't integrate correctly with quadlet's container lifecycle. A standalone `.service` file could bypass quadlet entirely for credential setup.

