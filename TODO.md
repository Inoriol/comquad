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

### Change Detection & Diffing (done)

- [x] **Reconcile package** — `internal/reconcile` with a directive-level 3-way merge (`MergeUnit`), read-only `Compute` / write `Apply` split, and unified-diff rendering.
- [x] **Baseline tracking** — pure-generated content stored under `$XDG_DATA_HOME/comquad/baseline/<project>/`; rewritten on `up`, removed on `down`, cleared by `regenerate`, rolled back on a failed `up`.
- [x] **Diff + confirmation on re-deploy** — `comquad up` shows a color-coded diff and asks for confirmation when a project is already deployed; `--no-diff` opts out.
- [x] **`--dry-run` diff** — shows new files as full content, changed files as a unified diff, and dropped services as removals.
- [x] **Selective restart** — only created/changed units are started/restarted; dropped services are stopped before `daemon-reload`.
- [x] **`down` verification** — verifies container units are stopped before removing files.
- [x] **Deterministic output** — sorted map iteration for labels, networks, and `depends_on` (was producing spurious diffs).

### Follow-ups

- **Value-level merge** — multi-value directives (`Environment=`, `Volume=`, `PublishPort=`) merge at key granularity; independent additions to the same key currently conflict instead of merging per-line.
- **Baseline rollback on failed `up`** — a failed deploy removes the baseline entirely (next `up` degrades to 2-way); consider restoring the previous baseline instead.
- **Diff rendering** — homegrown unified diff; could swap for `go-udiff` if richer output is ever needed.

### Future Security Improvements

- **Tmpfs-backed secrets via `LoadCredential`** — Currently secrets are bind-mounted directly from managed files on disk. Consider generating a companion `.service` service unit file (not a `.container` quadlet) that uses systemd `LoadCredential=` in `[Service]` combined with `Volume=%d/<name>` to mount secrets from systemd's RAM-backed credential directories (`/run/credentials/`). This would keep secret values in tmpfs memory rather than on persistent storage. Initial implementation attempted this using quadlet's `[Service]` pass-through, but `LoadCredential=` + `Volume=` with credential paths didn't integrate correctly with quadlet's container lifecycle. A standalone `.service` file could bypass quadlet entirely for credential setup.

