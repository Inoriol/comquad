//go:build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"comquad/tests/integration/helpers"
)

func TestEdit_WithFileModifications(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "up", "--name", project)

	unitName := fmt.Sprintf("cq-%s-web.service", project)
	helpers.AssertUnitActive(t, unitName, false)

	// Use sed as the editor to change "Image=" to "Image=changed-" in the quadlet file.
	// This causes a detectable modification, triggering daemon-reload + restart.
	oldEditor := os.Getenv("EDITOR")
	os.Setenv("EDITOR", "sed -i s/Image=/Image=changed-/")
	defer func() { os.Setenv("EDITOR", oldEditor) }()

	result := helpers.Comquad(t, dir, "edit", "--name", project, "web")
	if result.ExitCode != 0 {
		t.Fatalf("edit failed (exit=%d): stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	// Verify the unit file was modified on disk
	quadletFile := fmt.Sprintf("/etc/containers/systemd/cq-%s-web.container", project)
	content, err := os.ReadFile(quadletFile)
	if err != nil {
		t.Fatalf("failed to read quadlet file: %v", err)
	}
	if !strings.Contains(string(content), "Image=changed-") {
		t.Errorf("expected 'Image=changed-' in quadlet file, got:\n%s", string(content))
	}

	// Unit should still be active after edit+reload+restart
	helpers.AssertUnitActive(t, unitName, false)
}
