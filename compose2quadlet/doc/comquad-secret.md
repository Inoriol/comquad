## NAME

comquad-secret — comquad pseudo-quadlet secret unit

## SYNOPSIS

*name*.cq-secret

## DESCRIPTION

The `.cq-secret` unit type is a comquad-specific pseudo-quadlet that manages secrets using Podman's native secret store. Secret values are created as Podman secrets and mounted into containers at `/run/secrets/<secret-name>`.

`.cq-secret` units are not native Quadlet types. During deployment, comquad converts each `.cq-secret` file into a companion `.service` unit that systemd loads directly. The `.cq-secret` file remains the source of truth managed by comquad; the `.service` file is what systemd actually starts.

Secrets are generated from Compose file `secrets:` definitions. The conversion runs as a pre-mapping interceptor before container field mapping, producing both the `.cq-secret` unit and the corresponding `Secret=` directive on the consuming container.

### Secret Sources

Two source types are supported:

- **Environment secrets** — The secret value is read from a host environment variable at deployment time. The value is written to a temporary file in `/run/comquad/secrets/` (mode 0600), and the generated service uses `podman secret create` to load it into Podman's secret store.

- **File secrets** — The secret value is read from a file on the host filesystem. The generated service uses `podman secret create <name> <path>` to load the file contents into Podman's secret store.

### External Secrets

Secrets defined with `external: true` in the Compose file do not produce `.cq-secret` units. Instead, they generate a native Podman `Secret=<name>` directive on the container. External secrets require Podman 4.5.0 or newer.

### Custom Permissions Fallback

Secrets with custom `uid`, `gid`, or `mode` in the Compose service reference bypass the `.cq-secret` mechanism entirely. They fall back to a direct `Volume=` bind mount from the source file to the container target path. For environment secrets with custom permissions, the `WithSecretsDirectory()` option must be configured; the environment variable value is written to `<SecretsDir>/<secret_name>` (mode 0600) and mounted as a volume.

### Deduplication

When multiple services reference the same secret, only one `.cq-secret` unit is generated. Each consuming container receives its own `Secret=` directive referencing the same Podman secret. The project state tracks the deduplicated secret count.

### Secret Cleanup

When a project is removed with `comquad down`, the generated `.service` files include an `ExecStop=-/usr/bin/podman secret rm <name>` directive that removes the secret from Podman's secret store. The `-` prefix ensures the stop operation succeeds even if the secret doesn't exist.

## SECRET UNIT SECTIONS

### [Secret]

The `[Secret]` section describes the secret source. It is the only custom section in a `.cq-secret` unit. After pseudo-quadlet conversion, this section is consumed and replaced with standard systemd sections.

| Directive | Description |
|---|---|
| `Source=` | Source type: `environment` or `file` |
| `Environment=` | Name of the host environment variable (when `Source=environment`) |
| `Path=` | Absolute path to the secret file on the host (when `Source=file`) |

#### `Source=`

Required. Specifies how the secret value is obtained.

- `environment` — Read from a host environment variable. Requires `Environment=` to specify which variable.
- `file` — Read from a file on the host filesystem. Requires `Path=` to specify the file location.

#### `Environment=`

The name of the host environment variable to read the secret value from. Only valid when `Source=environment`.

During deployment, comquad reads the environment variable and writes its value to a temporary file at `/run/comquad/secrets/<project>-<secret-name>.env` (mode 0600). The generated service then loads this file into Podman's secret store:

```
ExecStart=/bin/sh -c 'podman secret create --replace <name> /run/comquad/secrets/<name>.env'
```

If the environment variable is not set or is empty at deployment time, no secret file is created and the service will fail to start.

#### `Path=`

The absolute path to the secret file on the host. Only valid when `Source=file`.

The generated service loads the file contents directly into Podman's secret store:

```
ExecStart=/usr/bin/podman secret create --replace <name> <path>
```

The file must exist and be readable by the systemd service at start time.

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
ExecStart=/bin/sh -c 'podman secret create --replace db_password /run/comquad/secrets/db_password.env'
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
  - `Secret=cq-myapp-api_key`
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

### Custom Permissions Fallback

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
- No `.cq-secret` unit
- Direct bind mount: `Volume=./db_password.txt:/run/secrets/db_password:ro`

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

## FILE LOCATIONS

`.cq-secret` files are stored in the systemd Quadlet directory alongside other generated units:

- Rootless: `~/.config/containers/systemd/`
- Rootful: `/etc/containers/systemd/`

The companion `.service` file is written to the systemd unit directory by comquad during deployment:

- Rootless: `~/.config/systemd/user/`
- Rootful: `/etc/systemd/system/`

For environment secrets, temporary secret files are written to `/run/comquad/secrets/` (mode 0700) during deployment. These files are read by the secret service to create the Podman secret and are not automatically cleaned up.

## SEE ALSO

- [podman-systemd.unit(5)](https://docs.podman.io/en/v6.1.0/markdown/podman-systemd.unit.5.html) — Native Quadlet unit types 
- [podman-secret(1)](https://docs.podman.io/en/latest/markdown/podman-secret.1.html) — Podman secret management commands
