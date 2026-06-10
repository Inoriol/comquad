# comquad

ComQuad (Compose + Quadlet 🍊) is a Docker-compose-like CLI for Podman Quadlets, backed by systemd.

`comquad` lets you define your services in a `compose` yaml file and deploy them as individual systemd units using Podman's Quadlet technology. Instead of running its own orchestrator, comquad prepares the quadlet files and delegates lifecycle management entirely to systemd.

## 🚧 Project Status: Infra-Built Utility

I am an infrastructure engineer, not a full-time software developer. I built **Comquad** to solve a specific problem for my own workflow.

- **Contributions:** I am currently not accepting complex feature pull requests because I do not have the bandwidth or Go expertise to maintain them.
- **Bugs:** Feel free to open issues if a specific Docker Compose file breaks, but fixes will happen on a "best effort" timeline.
- **Philosophy:** This tool is intentionally small, simple, and transparent. It is not trying to become Kubernetes.

## 🛠️ Requirements

- **Podman** (for `podman pull`)
- **podlet** (for transpiling `compose` yaml into quadlet files)
- **systemd** with quadlet support
- Go 1.23+ (to build from source)

## 🚀 Installation

Build from source:
```bash
go build -o comquad ./cmd/comquad/
sudo cp comquad /usr/local/bin/

```

Or install directly via Go:

```bash
go install comquad/cmd/comquad@latest

```

## ⚙️ Usage

### Deploy a project

From a directory containing `compose` yaml:

```bash
comquad up

```

* **Override project name:** `comquad up -n my-service`
* **Force rebuild all images:** `comquad up --build`
* **Control image pulls:** `comquad up --pull [always|missing|never]` *(default: missing)*
* **Rootless port offset:** Set `COMQUAD_PORT_OFFSET` env variable (default 2000) to shift privileged ports (< 1024) for rootless mode

### Manage & Monitor

* **Remove a project:** `comquad down`
* **List deployed projects:** `comquad list`
* **Show unit status:** `comquad ps`
* **Check prerequisites:** `comquad check`

### Stream logs

```bash
comquad logs                  # all services (one-shot)
comquad logs -f               # all services (follow)
comquad logs web              # single service
comquad logs -n myapp web db  # specific project/services

```

## 🏗️ System Architecture & Internal Mechanics

For a deep dive into how `comquad` processes your compose files, manages state, and maps directories, please see dedicated [Architecture Guide](./ARCHITECTURE.md) guide.

## 📄 License

MIT
