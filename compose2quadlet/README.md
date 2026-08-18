# compose2quadlet

Go library that transpiles Docker Compose files into Podman Quadlet unit files.

**Not a CLI tool.** This is a library designed to be consumed by [comquad](https://github.com/Inoriol/comquad) and other tools that need programmatic compose-to-quadlet conversion.

Heavily inspired by [podlet](https://github.com/containers/podlet), but designed for the Go ecosystem.

## Motivation

Replaces the former `podlet` binary dependency with a native Go library. Eliminates the fragile text-based pipeline (preprocess → strip → transpile → cook → graft) with structured, type-safe quadlet output.

## Usage

**Quick path** — `TranspileFile` loads the compose file with `.env` resolution and transpiles in one call:

```go
import (
    c2q "github.com/Inoriol/comquad/compose2quadlet"
    "github.com/Inoriol/comquad/compose2quadlet/serialization"
)

func main() {
    units, _ := c2q.TranspileFile("compose.yaml",
        c2q.WithProjectName("myapp"),
        c2q.WithPortOffset(10000),
        c2q.WithAutoUpdate(),
        c2q.WithLabels(map[string]string{
            "com.myorg.managed": "true",
            "com.myorg.project": "myapp",
        }),
    )

    // Write all units to a directory
    serialization.WriteUnits("/etc/containers/systemd", units)
}
```

**Full control** — load the compose-go project yourself and call `Transpile`:

```go
import (
    c2q "github.com/Inoriol/comquad/compose2quadlet"
    "github.com/Inoriol/comquad/compose2quadlet/serialization"
    "github.com/compose-spec/compose-go/v2/cli"
)

func main() {
    opts, _ := cli.NewProjectOptions(
        []string{"compose.yaml"},
        cli.WithOsEnv,
        cli.WithDotEnv,
    )
    project, _ := opts.LoadProject(context.Background())

    units, _ := c2q.Transpile(project,
        c2q.WithProjectName("myapp"),
        c2q.WithPortOffset(10000),
        c2q.WithAutoUpdate(),
        c2q.WithLabels(map[string]string{
            "com.myorg.managed": "true",
            "com.myorg.project": "myapp",
        }),
    )

    for _, u := range units {
        fmt.Printf("%s.%s\n", u.Name, u.Type)
    }

    // Serialize to ini format for writing to disk
    text := serialization.Marshal(units[0])
    fmt.Println(text)
}
```

## Packages

- **Root** — `Transpile()`, `TranspileFile()`, `ParseVersion()`, core types (`QuadletUnit`, `Section`, `Directive`), option constructors
- **`mapper/`** — Field mapping: compose-go `ServiceConfig` / `*Project` → quadlet directives and units
  - `container.go`, `healthcheck.go`, `security.go`, `ports.go` — per-service container directives
  - `service.go` — systemd `[Service]` resource-control and restart directives
  - `unit.go` — `depends_on` → `[Unit]` dependencies and health polling hooks
  - `image.go`, `build.go` — companion `.image` and `.build` structural units
  - `network.go`, `volume.go` — top-level `.network` and `.volume` structural units
  - `secrets.go` — pre-mapping interceptor for secrets and configs
  - `dockerfile.go` — `PatchDockerfileFROM()` normalizes bare image names in Dockerfile FROM lines
- **`opinionated/`** — Composable post-processing transforms (prefix, references including external resources, container name, SELinux relabeling, labels, default network, network aliases, port offset, auto-update, install section)
- **`serialization/`** — `Marshal()`, `Write()`, `WriteUnits()`, `Unmarshal()` for ini-format serialization

Warnings from degraded or skipped mappings are sent through the `WithInfo` callback when configured. Fatal mappings still return an error. Environment-secret and patched-Dockerfile write failures are reported as degraded warnings.

## Dependencies

- `github.com/compose-spec/compose-go/v2` — canonical compose parsing
- Standard library only otherwise

## Determinism

Directive ordering is deterministic: all map iteration (labels, annotations, environment, sysctls, service networks, `depends_on`, logging options, driver opts, etc.) is sorted, so transpiling the same compose file always produces byte-identical output. This makes the library safe for golden-file testing and reproducible deploys.

## License

MIT
