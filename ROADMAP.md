# Comquad Roadmap 

This document outlines the planned evolution of Comquad. Because I believe in the Unix philosophy—*do one thing and do it right*—our goal is to move from a tactical pre-compiler to a pure, lean systemd context orchestrator.

## 📍 Current Horizon: v0.1.0 (The Launchpad)
*Focus: Rock-solid local container state execution.*
- [x] Strict generation of `.container` files.
- [x] Forced creation of `.volume` units to guarantee data lifecycles.
- [x] Forced creation of `.network` units to ensure inter-container DNS resolution.
- [x] Temporary Go-based preprocessing engine to handle basic local `build:` blocks.

## 🚀 Next Horizon: v0.1.x - v0.5.0 (Context & Secrets)
*Focus: Eliminating friction points and handling advanced Compose features.*
- [ ] **Bypass Engine Layer:** Implement in-memory state tracking to safely strip, hold, and inject configuration blocks that upstream tools don't natively map yet.
- [ ] **Native Secrets Management:** Intercept Compose `secrets:` and inject them cleanly as Podman `Secret=` systemd keys.

## 🌎 Super long in future (possibly never)
- **Have a swarm compatability** Implement support for Docker Swarm compose syntex, utilizing `Eclipse BlueChi`.


## Regarding current Build Stage

### Current Implementation (The Temporary Bridge)
In the current release, Comquad acts as a tactical pre-compiler for Docker Compose `build:` blocks. 

Because current `podlet` has brittle edge cases when mapping complex local build structures to systemd, Comquad intercepts the `build:` metadata during the preprocessing phase. It orchestrates a local build on the host, replaces the `build:` configuration block with a deterministic local `image:` reference, and feeds clean YAML downstream. 

While this guarantees a seamless developer experience for the initial release, **this approach is explicitly designated as technical debt to be abolished in future versions.**

### The Future State: Pure Systemd-Driven Compilation
To preserve the Unix philosophy— *"Do one thing and do it right"* —Comquad will deprecate host-side build interception once `podlet` mature to fully support all Quadlet `.build` unit parameters. 

Instead of duplicating compilation logic inside the Go binary, Comquad will elevate its role to a pure **Systemd Context Orchestrator**. The future architecture will handle builds entirely through the native OS layer by adhering to the following cycle:

```text
               [ Future Build Workflow ]
                          │
                          ▼
             1. Mutate Relative Paths
     (Convert context paths to absolute strings)
                          │
                          ▼
              2. Delegate to Quadlet
       (Let upstream emit the .build units)
                          │
                          ▼
            3. Trigger Native Execution
  ("systemctl start <service>-build.service")
                          │
                          ▼
             4. Stream Active Logs
     ("journalctl -u <service>-build.service -f")

```

### Architectural Benefits of the Future State

1. **Zero Runtime Duplication:** Comquad avoids becoming a bloated container manager. It lets Podman and systemd manage the execution pipeline, process signaling, and cache boundaries.
2. **Deterministic Declarative State:** The entire infrastructure—from source compilation to network topology—remains declared as standalone systemd artifacts on the host OS.
3. **Interactive Developer Experience:** By attaching Comquad's terminal output directly to the active `journald` log stream during the oneshot build service execution, developers retain the real-time, interactive feedback loop of a traditional `compose build` command.
