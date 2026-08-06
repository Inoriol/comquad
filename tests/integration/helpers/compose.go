//go:build integration

package helpers

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

// WriteCompose writes a compose.yaml to a temp directory and returns
// the directory path and the project name declared in the compose file.
// The directory is automatically removed when the test ends.
func WriteCompose(t *testing.T, content string) (string, string) {
    t.Helper()
    dir := t.TempDir()
    path := filepath.Join(dir, "compose.yaml")
    if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
        t.Fatalf("failed to write compose.yaml: %v", err)
    }
    return dir, extractProjectName(content)
}

// extractProjectName parses the project name from a compose file string.
// It reads the top-level "name:" field, falling back to "unknown" if absent.
func extractProjectName(content string) string {
    for _, line := range strings.Split(content, "\n") {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "name:") {
            name := strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
            if name != "" {
                return name
            }
        }
    }
    return "unknown"
}

// WriteFile writes an arbitrary file into an existing directory.
// Useful for placing Dockerfiles next to compose.yaml.
func WriteFile(t *testing.T, dir, name, content string) {
    t.Helper()
    if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
        t.Fatalf("failed to write %s: %v", name, err)
    }
}

// --- Reusable compose templates ---

// SimpleCompose returns a minimal single-service compose with a named project.
// The project name is injected so each test gets isolated unit names.
func SimpleCompose(project string) string {
    return `name: ` + project + `
services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "8080:80"
`
}

// MultiServiceCompose returns a two-service compose sharing a network.
func MultiServiceCompose(project string) string {
    return `name: ` + project + `
services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "8081:80"
    depends_on:
      - api
  api:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
`
}

// WithVolumeCompose returns a compose with a named volume.
func WithVolumeCompose(project string) string {
	return `name: ` + project + `
services:
  db:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    volumes:
      - dbdata:/data
volumes:
  dbdata:
`
}

// MultiNetworkCompose returns a compose with two services on different
// networks — they should not be able to resolve each other.
func MultiNetworkCompose(project string) string {
	return `name: ` + project + `
services:
  alpha:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    networks:
      - alpha-net
  beta:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    networks:
      - beta-net
networks:
  alpha-net:
  beta-net:
`
}

// FailingCompose returns a compose with a service that will fail to start
// because the command exits immediately with a non-zero code.
func FailingCompose(project string, image string) string {
	return `name: ` + project + `
services:
  failer:
    image: ` + image + `
    command: ["sh", "-c", "exit 1"]
`
}
