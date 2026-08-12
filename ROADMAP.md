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
- [x] **Graft Handlers:** Implement handlers in `internal/graft/handlers/` to safely strip, hold, and inject configuration blocks that upstream tools don't natively map yet (e.g. skipping registry pulls for podlet-generated build images).
- [x] **Native Secrets Management:** Intercept Compose `secrets:` and inject them cleanly as Podman `Secret=` systemd keys.
- [x] **Native go implementation of compose2quadlet** Replaced podlet with the compose2quadlet Go library for all compose→quadlet transpilation.

## 🌎 Super long in future (possibly never)
- **Have a swarm compatability** Implement support for Docker Swarm compose syntex, utilizing `Eclipse BlueChi`.

