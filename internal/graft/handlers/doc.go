// Package handlers provides pluggable post-processing handlers for the Graft pipeline step.
//
// Each handler addresses a specific gap between Docker Compose syntax and Quadlet file format
// that podlet does not support natively. Handlers are registered with the Grafter and executed
// sequentially during the Process step.
//
// Planned handlers:
//   - build: intercept build blocks to skip registry pulls on locally-built images
//   - secrets: translate compose secrets into Podman Secret= quadlet keys
package handlers
