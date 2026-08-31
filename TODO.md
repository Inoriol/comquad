## 🗺️ Roadmap & Next Steps

For long-term goals, refer to the [roadmap](./ROADMAP.md).

### Follow-ups

- **Value-level merge** — Multi-value directives (`Environment=`, `Volume=`, `PublishPort=`) are merged at key granularity; independent additions to the same key currently conflict instead of being merged per line.
- **Diff rendering** — The unified diff implementation is homegrown; it could be replaced with `go-udiff` if richer output is ever needed.

### Possible Security Improvements

- **Tmpfs-backed secrets via `LoadCredential`** — Currently, secrets are bind-mounted directly from managed files on disk. Consider generating a companion `.service` unit (rather than a `.container` Quadlet) that uses systemd `LoadCredential=` in `[Service]`, combined with `Volume=%d/<name>`, to mount secrets from systemd's RAM-backed credential directories (`/run/credentials/`). This would keep secret values in tmpfs memory rather than on persistent storage. An initial implementation using Quadlet's `[Service]` pass-through did not integrate `LoadCredential=` and `Volume=` with credential paths correctly in the container lifecycle. A standalone `.service` file could bypass Quadlet entirely for credential setup.

### Image and Build Handling Improvements

- **Using Quadlet for pre-pulling** — Instead of pulling images directly, create and start `.image` Quadlets and check for errors.
- **Using Quadlet for builds** — Consider adding a `comquad build` command with similar logic.

### Quadlet Directives in Compose Files

- **X-extension** — Consider how to incorporate Quadlet options that the Compose specification does not support, perhaps through an X-extension.
- **Questionable future feature: Round-trip** — Make it possible to convert a project back into a Compose file.
