//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/Inoriol/comquad/tests/integration/helpers"
)

func TestExec_AmbiguousService_Errors(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)

	compose := `name: ` + project + `
services:
  alpha:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    container_name: shared
  beta:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    container_name: shared
`
	dir, _ := helpers.WriteCompose(t, compose)

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "up", "--name", project)

	// "shared" matches both alpha and beta via ContainerName= directive
	result := helpers.MustFail(t, dir, "exec",
		"--name", project,
		"--tty=false",
		"shared",
		"--", "echo", "hi",
	)

	if !strings.Contains(result.Stderr, "matched multiple containers") &&
		!strings.Contains(result.Stdout, "matched multiple containers") {
		t.Fatalf("expected ambiguous match error, got:\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
	}
}
