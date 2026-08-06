# Comquad Roadmap 

This document outlines the planned evolution of Comquad. Because I believe in the Unix philosophy—*do one thing and do it right*—our goal is to move from a tactical pre-compiler to a pure, lean systemd context orchestrator.

## 📍 Current Horizon: v0.1.0 (The Launchpad)
*Focus: Rock-solid local container state execution.*
- [x] Strict generation of `.container` files.
- [x] Forced creation of `.volume` units to guarantee data lifecycles.
- [x] Forced creation of `.network` units to ensure inter-container DNS resolution.
- [x] Pass `build:` blocks through to podlet unchanged for native handling.

## 🚀 Next Horizon: v0.1.x - v0.5.0 (Context & Secrets)
*Focus: Eliminating friction points and handling advanced Compose features.*
- [ ] **Graft Handlers:** Implement handlers in `internal/graft/handlers/` to safely strip, hold, and inject configuration blocks that upstream tools don't natively map yet (e.g. skipping registry pulls for podlet-generated build images).
- [ ] **Native Secrets Management:** Intercept Compose `secrets:` and inject them cleanly as Podman `Secret=` systemd keys.

## 🌎 Super long in future (possibly never)
- **Have a swarm compatability** Implement support for Docker Swarm compose syntex, utilizing `Eclipse BlueChi`.


## Regarding the Build Stage

### Current Implementation

`build:` blocks in compose files are passed through to podlet unchanged. Podlet handles `build:` natively by generating `.build` quadlet files. Comquad no longer intercepts, replaces, or pre-builds `build:` blocks.

### The Future State: Pure Systemd-Driven Compilation

Future `graft/handlers/` will address gaps between what podlet emits and what a smooth developer experience requires (e.g. detecting build-generated images and skipping registry pulls). The goal remains the same: comquad as a pure systemd context orchestrator, with podlet and systemd managing the build execution pipeline.
