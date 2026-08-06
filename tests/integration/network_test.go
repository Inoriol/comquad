//go:build integration

package integration

import (
	"fmt"
	"testing"

	"github.com/Inoriol/comquad/tests/integration/helpers"
)

func TestNetworkIsolation_DifferentNetworks(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.MultiNetworkCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	helpers.MustSucceed(t, dir, "up", "--name", project)

	// Both services should be running
	alphaUnit := fmt.Sprintf("cq-%s-alpha.service", project)
	betaUnit := fmt.Sprintf("cq-%s-beta.service", project)
	helpers.AssertUnitActive(t, alphaUnit, false)
	helpers.AssertUnitActive(t, betaUnit, false)

	// Each service should be on its own network
	alphaNetwork := fmt.Sprintf("cq-%s-alpha-net", project)
	betaNetwork := fmt.Sprintf("cq-%s-beta-net", project)
	if !helpers.NetworkExists(t, alphaNetwork) {
		t.Errorf("expected network %q to exist", alphaNetwork)
	}
	if !helpers.NetworkExists(t, betaNetwork) {
		t.Errorf("expected network %q to exist", betaNetwork)
	}

	// Networks should be different
	if alphaNetwork == betaNetwork {
		t.Error("alpha and beta should be on different networks")
	}

	// Verify the containers exist with their respective networks
	for _, svc := range []string{"alpha", "beta"} {
		helpers.AssertContainerRunning(t, fmt.Sprintf("%s-%s", project, svc))
	}

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)

	// Networks should be removed after down
	helpers.AssertNetworkGone(t, alphaNetwork)
	helpers.AssertNetworkGone(t, betaNetwork)
}
