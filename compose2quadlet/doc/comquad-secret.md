## NAME

comquad-secret — comquad pseudo-quadlet secret unit

## SYNOPSIS

*name*.cq-secret

## DESCRIPTION

The `.cq-secret` unit type is a comquad-specific pseudo-quadlet that manages secrets using Podman's native secret store. Secret values are created as Podman secrets and mounted into containers at `/run/secrets/<secret-name>` or exposed as environment variables.

`.cq-secret` units are not native Quadlet types. During deployment, comquad converts each `.cq-secret` file into a companion `.service` unit that systemd loads directly. The `.cq-secret` file remains the source of truth managed by comquad; the `.service` file is what systemd actually starts.

Secrets are generated from Compose file `secrets:` definitions. The conversion runs as a pre-mapping interceptor before container field mapping, producing both the `.cq-secret` unit and the corresponding `Secret=` directive on the consuming container.

### Secret Sources

Two source types are supported:

- **Environment secrets** — The secret value is read from a host environment variable at deployment time. The value is written to a temporary file in the runtime secrets directory (mode 0600), and the generated service uses `podman secret create` to load it into Podman's secret store.

- **File secrets** — The secret value is read from a file on the host filesystem. The generated service uses `podman secret create <name> <path>` to load the file contents into Podman's secret store.

### Secret Options

Secrets support the following options passed to the `Secret=` directive:

- **target=** — Custom mount path inside the container. Relative paths are resolved relative to `/run/secrets/`. For environment variable exposure, this is the variable name.
- **uid=** — UID of the secret file (mount type only).
- **gid=** — GID of the secret file (mount type only).
- **mode=** — File mode of the secret file (mount type only).

### Secret Drivers

Secrets support Podman's secret driver mechanism via the `driver` and `driver_opts` fields in the Compose file. These correspond to `podman secret create --driver` and `--driver-opts`.

- **driver** — The secret driver to use (e.g., `file`, `pass`, `shell`). Defaults to `file`.
- **driver_opts** — Driver-specific options as key-value pairs.

Example with the `pass` driver for GPG-encrypted secrets:
```yaml
secrets:
  my_encrypted_secret:
    file: ./secret.txt.gpg
    driver: pass
```

Example with the `shell` driver for script-managed secrets:
```yaml
secrets:
  my_shell_secret:
    driver: shell
    driver_opts:
      list: /usr/local/bin/list.sh
      lookup: /usr/local/bin/lookup.sh
      store: /usr/local/bin/store.sh
      delete: /usr/local/bin/delete.sh
```

When using the `shell` driver without a `file` or `environment` field, the secret is managed entirely by the driver scripts. The generated service reads from stdin (`-`) when creating the secret.

### Environment Variable Exposure

Secrets can be exposed as environment variables instead of files using the `x-type: env` extension:

```yaml
services:
  web:
    secrets:
      - source: db_password
        x-type: env
        x-target: DB_PASSWORD  # optional, defaults to secret name
```

This generates `Secret=<name>,type=env,target=DB_PASSWORD` which exposes the secret as an environment variable inside the container.

**Note:** `x-type: env` is incompatible with `uid`, `gid`, and `mode` options. If both are specified, a warning is emitted and the options are ignored.

### External Secrets

Secrets defined with `external: true` in the Compose file do not produce `.cq-secret` units. Instead, they generate a native Podman `Secret=<name>` directive on the container. External secrets require Podman 4.5.0 or newer.

External secrets support the same options (target, uid, gid, mode) and x-extensions (x-type, x-target) as managed secrets.

### Version Compatibility

- **Podman >= 4.5.0**: Uses native `Secret=` directive with all options (target, uid, gid, mode, type).
- **Podman < 4.5.0**: Falls back to `Volume=` bind mounts for secrets. Environment secrets require `WithSecretsDirectory()` to be configured.

### Custom Permissions

Secrets with custom `uid`, `gid`, or `mode` are passed directly to the `Secret=` directive:

```yaml
services:
  web:
    secrets:
      - source: db_password
        uid: "1000"
        gid: "1000"
        mode: 0400
```

Generates: `Secret=cq-myapp-db_password,uid=1000,gid=1000,mode=0400`

### Deduplication

When multiple services reference the same secret, only one `.cq-secret` unit is generated. Each consuming container receives its own `Secret=` directive referencing the same Podman secret. The project state tracks the deduplicated secret count.

### Secret Cleanup

When a project is removed with `comquad down`, the generated `.service` files include an `ExecStop=-/usr/bin/podman secret rm <name>` directive that removes the secret from Podman's secret store. The `-` prefix ensures the stop operation succeeds even if the secret doesn't exist.

## SECRET UNIT SECTIONS

### [Secret]

The `[Secret]` section describes the secret source. It is the only custom section in a `.cq-secret` unit. After pseudo-quadlet conversion, this section is consumed and replaced with standard systemd sections.

| Directive | Description |
|---|---|
| `Source=` | Source type: `environment`, `file`, or `driver` |
| `Environment=` | Name of the host environment variable (when `Source=environment`) |
| `Path=` | Absolute path to the secret file on the host (when `Source=file`) |
| `Driver=` | Secret driver name (e.g., `file`, `pass`, `shell`) |
| `DriverOpts=` | Comma-separated list of driver-specific options (`key=value,...`) |

#### `Source=`

Required. Specifies how the secret value is obtained.

- `environment` — Read from a host environment variable. Requires `Environment=` to specify which variable.
- `file` — Read from a file on the host filesystem. Requires `Path=` to specify the file location.
- `driver` — Managed by a secret driver. Requires `Driver=` and optionally `DriverOpts=`. Used when neither `file` nor `environment` is specified but a driver is configured.

#### `Environment=`

The name of the host environment variable to read the secret value from. Only valid when `Source=environment`.

During deployment, comquad reads the environment variable and writes its value to a temporary file at `<runtime-secrets-dir>/<name>.env` (mode 0600). The runtime secrets directory is `$XDG_RUNTIME_DIR/comquad/secrets/` for rootless mode (default: `/run/user/<uid>/comquad/secrets/`) or `/run/comquad/secrets/` for rootful mode. The generated service then loads this file into Podman's secret store:

```
ExecStart=/bin/sh -c '/usr/bin/podman secret create --replace <name> <runtime-secrets-dir>/<name>.env'
```

If the environment variable is not set or is empty at deployment time, no secret file is created and the service will fail to start.

#### `Path=`

The absolute path to the secret file on the host. Only valid when `Source=file`.

The generated service loads the file contents directly into Podman's secret store:

```
ExecStart=/usr/bin/podman secret create --replace <name> <path>
```

The file must exist and be readable by the systemd service at start time.

#### `Driver=`

The secret driver to use for storing the secret. Corresponds to `podman secret create --driver`. Supported drivers include:

- `file` — Secret resides in a read-protected file (default).
- `pass` — Secret resides in a GPG-encrypted file.
- `shell` — Secret is managed by custom scripts.

#### `DriverOpts=`

Comma-separated list of driver-specific options. Corresponds to `podman secret create --driver-opts`. Used primarily with the `shell` driver to specify script paths:

```
DriverOpts=list=/usr/local/bin/list.sh,lookup=/usr/local/bin/lookup.sh
```

## GENERATED SERVICE STRUCTURE

After pseudo-quadlet conversion, each `.cq-secret` unit produces a `.service` file with the following structure:

### Environment Secret

For a `.cq-secret` unit:

```ini
[Secret]
Source=environment
Environment=DB_PASSWORD
```

The generated `.service` file is:

```ini
[Unit]
Description=comquad secret: db_password

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/sh -c '/usr/bin/podman secret create --replace db_password <runtime-secrets-dir>/db_password.env'
ExecStop=-/usr/bin/podman secret rm db_password

[Install]
WantedBy=default.target
```

### File Secret

For a `.cq-secret` unit:

```ini
[Secret]
Source=file
Path=/etc/secrets/api.key
```

The generated `.service` file is:

```ini
[Unit]
Description=comquad secret: api_key

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/podman secret create --replace api_key /etc/secrets/api.key
ExecStop=-/usr/bin/podman secret rm api_key

[Install]
WantedBy=default.target
```

### File Secret with Driver

For a `.cq-secret` unit with the `pass` driver:

```ini
[Secret]
Source=file
Path=/etc/secrets/secret.txt.gpg
Driver=pass
```

The generated `.service` file is:

```ini
[Unit]
Description=comquad secret: encrypted_secret

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/podman secret create --driver=pass --replace encrypted_secret /etc/secrets/secret.txt.gpg
ExecStop=-/usr/bin/podman secret rm encrypted_secret

[Install]
WantedBy=default.target
```

### Driver-Only Secret (shell driver)

For a `.cq-secret` unit with the `shell` driver and no file or environment:

```ini
[Secret]
Source=driver
Driver=shell
DriverOpts=list=/usr/local/bin/list.sh,lookup=/usr/local/bin/lookup.sh,store=/usr/local/bin/store.sh,delete=/usr/local/bin/delete.sh
```

The generated `.service` file is:

```ini
[Unit]
Description=comquad secret: shell_secret

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/sh -c '/usr/bin/podman secret create --driver=shell --driver-opts=list=/usr/local/bin/list.sh,lookup=/usr/local/bin/lookup.sh,store=/usr/local/bin/store.sh,delete=/usr/local/bin/delete.sh --replace shell_secret -'
ExecStop=-/usr/bin/podman secret rm shell_secret

[Install]
WantedBy=default.target
```

### Service Characteristics

- **Type=oneshot** — The service runs the secret creation command and exits.
- **RemainAfterExit=yes** — The service is considered active after the command completes, indicating the secret exists in Podman's store.
- **ExecStart=** — Creates the secret in Podman's secret store using `podman secret create --replace`. The `--replace` flag ensures secrets are updated when their values change.
- **ExecStop=** — Removes the secret from Podman's store when the service is stopped. The `-` prefix ensures the stop operation succeeds even if the secret doesn't exist.

## CONTAINER INTEGRATION

Containers that consume a secret receive a `Secret=` directive that mounts the secret from Podman's secret store into the container:

```
Secret=<secret_name>
```

Podman automatically mounts the secret at `/run/secrets/<secret_name>` inside the container with mode 0444 (read-only). The secret name in the container matches the Podman secret name, which includes the project prefix (e.g., `cq-myapp-db_password`).

### Service Naming Convention

The pseudo-quadlet service naming follows the native quadlet convention:

| Pseudo-quadlet file | Systemd service |
|---|---|
| `cq-myapp-db_password.cq-secret` | `cq-myapp-db_password-secret.service` |

The `-secret.service` suffix distinguishes secret services from other pseudo-quadlet types (`.cq-image` → `-image.service`, `.cq-build` → `-build.service`).

Container units reference the secret service via `After=` and `Requires=` directives in the `[Unit]` section, ensuring the secret is created before the container starts.

## COMPOSE FILE MAPPING

### Default Behavior (Podman secrets)

```yaml
services:
  web:
    image: nginx
    secrets:
      - db_password
      - source: api_key
        target: /app/config/api.key

secrets:
  db_password:
    environment: DB_PASSWORD
  api_key:
    file: ./api.key
```

Generates:
- Two `.cq-secret` units (`db_password` and `api_key`)
- Two `Secret=` directives on the `web` container:
  - `Secret=cq-myapp-db_password`
  - `Secret=cq-myapp-api_key,target=/app/config/api.key`
- Secrets are mounted at `/run/secrets/cq-myapp-db_password` and `/run/secrets/cq-myapp-api_key` in the container

### External Secrets

```yaml
services:
  web:
    image: nginx
    secrets:
      - db_password

secrets:
  db_password:
    external: true
    name: my-db-secret
```

Generates:
- No `.cq-secret` unit
- `Secret=my-db-secret` directive on the `web` container

### Custom Permissions

```yaml
services:
  web:
    image: nginx
    secrets:
      - source: db_password
        uid: "1000"
        gid: "1000"
        mode: 0400

secrets:
  db_password:
    file: ./db_password.txt
```

Generates:
- One `.cq-secret` unit (`db_password`)
- `Secret=cq-myapp-db_password,uid=1000,gid=1000,mode=0400` directive on the `web` container

### Environment Variable Exposure

```yaml
services:
  web:
    image: nginx
    secrets:
      - source: db_password
        x-type: env
        x-target: DB_PASSWORD

secrets:
  db_password:
    environment: DB_PASSWORD
```

Generates:
- One `.cq-secret` unit (`db_password`)
- `Secret=cq-myapp-db_password,type=env,target=DB_PASSWORD` directive on the `web` container
- The secret is exposed as the `DB_PASSWORD` environment variable inside the container

### Shared Secrets (Deduplication)

```yaml
services:
  web:
    image: nginx
    secrets:
      - shared_key
  api:
    image: node
    secrets:
      - shared_key

secrets:
  shared_key:
    environment: SHARED_KEY
```

Generates:
- One `.cq-secret` unit (`shared_key`) — deduplicated
- Two `Secret=` directives (one per container), both referencing the same Podman secret

### Secret Drivers

```yaml
services:
  web:
    image: nginx
    secrets:
      - encrypted_secret

secrets:
  encrypted_secret:
    file: ./secret.txt.gpg
    driver: pass
```

Generates:
- One `.cq-secret` unit (`encrypted_secret`) with `Driver=pass`
- `Secret=cq-myapp-encrypted_secret` directive on the `web` container
- The generated service uses `podman secret create --driver=pass`

### Shell Driver (Script-Managed Secrets)

```yaml
services:
  web:
    image: nginx
    secrets:
      - shell_secret

secrets:
  shell_secret:
    driver: shell
    driver_opts:
      list: /usr/local/bin/list.sh
      lookup: /usr/local/bin/lookup.sh
      store: /usr/local/bin/store.sh
      delete: /usr/local/bin/delete.sh
```

Generates:
- One `.cq-secret` unit (`shell_secret`) with `Source=driver`, `Driver=shell`, and `DriverOpts=...`
- `Secret=cq-myapp-shell_secret` directive on the `web` container
- The generated service uses `podman secret create --driver=shell --driver-opts=...` and reads from stdin

## FILE LOCATIONS

`.cq-secret` files are stored in the systemd Quadlet directory alongside other generated units:

- Rootless: `~/.config/containers/systemd/`
- Rootful: `/etc/containers/systemd/`

The companion `.service` file is written to the systemd unit directory by comquad during deployment:

- Rootless: `~/.config/systemd/user/`
- Rootful: `/etc/systemd/system/`

For environment secrets, temporary secret files are written to the runtime secrets directory (mode 0700) during deployment:

- Rootless: `$XDG_RUNTIME_DIR/comquad/secrets/` (default: `/run/user/<uid>/comquad/secrets/`)
- Rootful: `/run/comquad/secrets/`

These files are read by the secret service to create the Podman secret and are not automatically cleaned up.

## SEE ALSO

- [podman-systemd.unit(5)](https://docs.podman.io/en/v6.1.0/markdown/podman-systemd.unit.5.html) — Native Quadlet unit types 
- [podman-secret(1)](https://docs.podman.io/en/latest/markdown/podman-secret.1.html) — Podman secret management commands
