//go:build integration

package integration

import (
    "fmt"
    "strings"
    "testing"
    "time"

    "comquad/tests/integration/helpers"
)

func TestUpDown_SimpleService(t *testing.T) {
    project := helpers.ProjectName(t)
    dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

    // Always tear down, even if the test fails mid-way
    t.Cleanup(func() {
        helpers.Comquad(t, dir, "down", "--name", project)
    })

    // --- UP ---
    helpers.MustSucceed(t, dir, "up", "--name", project)

    unitName := fmt.Sprintf("cq-%s-web.service", project)
    helpers.AssertUnitActive(t, unitName, false)
    helpers.AssertContainerRunning(t, fmt.Sprintf("%s-web", project))

    state := helpers.AssertProjectRegistered(t, project)
    if len(state.Files) == 0 {
        t.Fatal("expected state.Files to be non-empty after up")
    }

    // --- DOWN ---
    helpers.MustSucceed(t, dir, "down")

    helpers.AssertUnitInactive(t, unitName, false)
    helpers.AssertContainerGone(t, fmt.Sprintf("%s-web", project))
    helpers.AssertProjectGone(t, project)

    networkName := fmt.Sprintf("cq-%s-default", project)
    helpers.AssertNetworkGone(t, networkName)
}

func TestUpDown_MultiService(t *testing.T) {
    project := helpers.ProjectName(t)
    dir, _ := helpers.WriteCompose(t, helpers.MultiServiceCompose(project))

    t.Cleanup(func() {
        helpers.Comquad(t, dir, "down", "--name", project)
    })

    helpers.MustSucceed(t, dir, "up", "--name", project)

    for _, svc := range []string{"web", "api"} {
        unit := fmt.Sprintf("cq-%s-%s.service", project, svc)
        helpers.AssertUnitActive(t, unit, false)
        helpers.AssertContainerRunning(t, fmt.Sprintf("%s-%s", project, svc))
    }

    helpers.MustSucceed(t, dir, "down")

    for _, svc := range []string{"web", "api"} {
        unit := fmt.Sprintf("cq-%s-%s.service", project, svc)
        helpers.AssertUnitInactive(t, unit, false)
    }
}

func TestDown_WithVolumes(t *testing.T) {
    project := helpers.ProjectName(t)
    dir, _ := helpers.WriteCompose(t, helpers.WithVolumeCompose(project))

    t.Cleanup(func() {
        helpers.Comquad(t, dir, "down", "--name", project)
    })

    helpers.MustSucceed(t, dir, "up", "--name", project)

    volumeName := fmt.Sprintf("cq-%s-dbdata", project)
    if !helpers.VolumeExists(t, volumeName) {
        t.Fatalf("expected volume %q to exist after up", volumeName)
    }

    // down WITHOUT -d should leave volume intact
    helpers.MustSucceed(t, dir, "down")
    if !helpers.VolumeExists(t, volumeName) {
        t.Fatal("volume should survive down without -d flag")
    }

    // now bring up again and down WITH -d
    helpers.MustSucceed(t, dir, "up")
    helpers.MustSucceed(t, dir, "down")
    if helpers.VolumeExists(t, volumeName) {
        t.Fatal("volume should be removed after down -d")
    }
}

func TestUp_Idempotent(t *testing.T) {
    project := helpers.ProjectName(t)
    dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

    t.Cleanup(func() {
        helpers.Comquad(t, dir, "down", "--name", project)
    })

    // First up
    helpers.MustSucceed(t, dir, "up", "--name", project)
    state1 := helpers.AssertProjectRegistered(t, project)

    // Second up on same project — should not error
    helpers.MustSucceed(t, dir, "up", "--name", project)
    state2 := helpers.AssertProjectRegistered(t, project)

    // Files list should be stable
    if len(state1.Files) != len(state2.Files) {
        t.Fatalf("file count changed between two ups: %d → %d",
            len(state1.Files), len(state2.Files))
    }
}

func TestUp_DryRun_NoSideEffects(t *testing.T) {
    project := helpers.ProjectName(t)
    dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

    // dry-run should never need cleanup — but add it defensively
    t.Cleanup(func() {
        helpers.Comquad(t, dir, "down", "--name", project)
    })

    result := helpers.MustSucceed(t, dir, "up", "--name", project, "--dry-run")

    // Output should mention the target path and file content
    if !strings.Contains(result.Stdout, "cq-"+project) {
        t.Fatalf("dry-run output missing expected project prefix, got:\n%s", result.Stdout)
    }

    // Nothing should be registered in state
    projects := helpers.ReadStateFile(t)
    if _, ok := projects[project]; ok {
        t.Fatal("dry-run must not register project in state file")
    }

    // No unit files should exist
    unitName := fmt.Sprintf("cq-%s-web.container", project)
    if helpers.UnitExists(t, unitName, false) {
        t.Fatal("dry-run must not create systemd unit files")
    }
}

func TestLifecycle_StartStopRestart(t *testing.T) {
    project := helpers.ProjectName(t)
    dir, _ := helpers.WriteCompose(t, helpers.SimpleCompose(project))

    t.Cleanup(func() {
        helpers.Comquad(t, dir, "down", "--name", project)
    })

    helpers.MustSucceed(t, dir, "up", "--name", project)

    unitName := fmt.Sprintf("cq-%s-web.service", project)
    helpers.AssertUnitActive(t, unitName, false)

    // stop
    helpers.MustSucceed(t, dir, "stop", "--name", project)
    helpers.AssertUnitInactive(t, unitName, false)

    // start
    helpers.MustSucceed(t, dir, "start", "--name", project)
    helpers.AssertUnitActive(t, unitName, false)

    // restart
    helpers.MustSucceed(t, dir, "restart", "--name", project)
    // give systemd a moment to cycle
    time.Sleep(2 * time.Second)
    helpers.AssertUnitActive(t, unitName, false)
}
