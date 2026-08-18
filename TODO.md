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
- [x] **Baseline rollback on failed `up`** — resolved: a failed deploy now restores the previous files and baseline (`rollbackDeploy`) instead of removing the baseline entirely.
- **Diff rendering** — homegrown unified diff; could swap for `go-udiff` if richer output is ever needed.

### Future Security Improvements

- **Tmpfs-backed secrets via `LoadCredential`** — Currently secrets are bind-mounted directly from managed files on disk. Consider generating a companion `.service` service unit file (not a `.container` quadlet) that uses systemd `LoadCredential=` in `[Service]` combined with `Volume=%d/<name>` to mount secrets from systemd's RAM-backed credential directories (`/run/credentials/`). This would keep secret values in tmpfs memory rather than on persistent storage. Initial implementation attempted this using quadlet's `[Service]` pass-through, but `LoadCredential=` + `Volume=` with credential paths didn't integrate correctly with quadlet's container lifecycle. A standalone `.service` file could bypass quadlet entirely for credential setup.

### Code Review Findings (2026-08-16) — resolved

Bugs and correctness issues found and fixed while analyzing the codebase:

- [x] **`comquad list` filters by cwd instead of listing everything** — `List()` now takes an explicit `filter` argument; the `list` command passes the raw `-n` value (empty when absent) instead of the cwd-derived project name.
- [x] **Prefix collision in file/unit discovery** — `collectProjectFiles`, `RegenerateState`, and `viewProject` now use the `"cq-" + projectName + "-"` prefix (with trailing `-`), matching `reconcile.Compute`/`registerState`.
- [x] **Podman version is never passed to compose2quadlet** — `Up` now calls `deploy.DetectPodmanVersion()` and passes `WithPodmanVersion()`, so version gates reflect the installed podman instead of defaulting to "latest". `ValidatePodmanVersion` was refactored to reuse the same detection.
- [x] **`comquad logs -f` only follows the first unit group** — follow mode now starts every `journalctl -f` group concurrently and merges the parsed streams into a single timestamp-ordered flush loop.
- [x] **`ps` D-Bus merge uses the wrong unit name** — `Ps` now queries `cq-<project>-<service>.service` instead of `<project>-<service>.service`.
- [x] **`resolveUnits` image/build match condition is broken** — the image/build match now mirrors `MatchQuadletResource` (both `name-<type>.service` and `name-<type>` forms).
- [x] **Destructive rollback on failed `up`** — `cleanup()` is now a `rollback()` that reverts only the files/baselines this deploy touched (using `plan` old contents) and restores the prior state, leaving the previous deployment intact.
- [x] **`regenerate` omits `Resources.Images`/`Builds`** — `RegenerateState` now populates `images`/`builds` from `.image`/`.build` filenames.
- [x] **Dead code** — removed `hasBuildFile`, `resolveContainerImages`, `normalizeImageRef`, `isRegistryWithPort` (orchestrator copy), and `truncate`, plus their tests.
- [x] **Doc** — corrected the ARCHITECTURE.md label claim: `.image` units carry no `Label=` directive and are identified by filename. (`restart: unless-stopped` was reviewed and left as-is: systemd `Restart=always` respects a manual `systemctl stop`, so the mapping is a reasonable approximation; `processConfig`'s `%04o` mode formatting is correct.)

### Code Review Findings (2026-08-18) — resolved

Bugs and correctness issues found while analyzing the codebase:

**High Severity:**
- [x] **`--since=` passed empty to journalctl in follow mode** — `buildJournalctlFollowCmd` now only adds `--since=` when `since != ""`. Affects `comquad logs -f` when `--since` is not provided (note: `comquad up -f` passes a valid deployment timestamp, so it is unaffected).
- [x] **`rollbackDeploy` unconditionally removes secrets directory** — Removed `os.RemoveAll(secretsDir)` from `rollbackDeploy`. Secrets dir cleanup already happens in `down`, so a failed deploy no longer destroys secrets from a prior successful deployment.

**Medium Severity:**
- [x] **Doc/Code mismatch: Default network attachment** — ARCHITECTURE.md updated to clarify that any container without a `Network=` directive is auto-attached to `cq-default`, even when user-defined networks exist.

**Low Severity:**
- [x] **`Regenerate` resource count omits Images and Builds** — `regenerate.go:37` now includes `Images` and `Builds` in the total resource count.
- [x] **Misleading comment in `parseContainer`** — Comment in `ps_podman.go:93-94` corrected to reflect that Podman container names are `<project>-<service>` (no `cq-` prefix).
- [x] **Unnecessary Orchestrator creation in `list` command** — `list.go` now directly uses `deploy.NewStateManager()` instead of creating an `Orchestrator`, avoiding an unnecessary `os.Getwd()` call.

