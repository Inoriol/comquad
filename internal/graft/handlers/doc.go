// Package handlers provides pluggable post-processing handlers for the Graft pipeline step.
//
// Each handler addresses a specific gap between Docker Compose syntax and Quadlet file format
// that podlet does not support natively. Handlers are registered with the Grafter and executed
// sequentially during the Process step.
//
// Implemented handlers:
//   - image: creates .image quadlet files for every .container, moving Image=, Policy=,
//     OS=, Arch=, and Variant= directives into dedicated image units so systemd can
//     manage image pulls separately.
//   - secret: translates compose secrets into quadlet directives. External secrets
//     produce Secret= for Podman's native secret store. File-based and environment-based
//     secrets produce LoadCredential= in [Service] and Volume= in [Container], mounting
//     secrets at /run/secrets/<name> via systemd credential directories.
//
// Planned handlers:
//   - build: intercept build blocks to skip registry pulls on locally-built images
package handlers
