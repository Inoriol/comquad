package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestNetworks_Basic(t *testing.T) {
	networks := types.Networks{
		"frontend": types.NetworkConfig{
			Driver: "bridge",
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Networks(networks, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 network unit, got %d", len(units))
	}
	u := units[0]
	if u.Type != c2qtypes.UnitNetwork || u.Name != "frontend" {
		t.Fatalf("expected frontend network unit, got %s/%s", u.Type, u.Name)
	}

	dirs := u.Sections[0].Directives
	if len(dirs) != 0 {
		t.Fatal("expected no directives for driver=bridge (default)")
	}
}

func TestNetworks_CustomDriver(t *testing.T) {
	networks := types.Networks{
		"mynet": types.NetworkConfig{
			Driver: "macvlan",
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Networks(networks, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Driver", "macvlan")
}

func TestNetworks_DriverOpts(t *testing.T) {
	networks := types.Networks{
		"mynet": types.NetworkConfig{
			DriverOpts: types.Options{"mtu": "1400", "isolate": "true"},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Networks(networks, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Options", "mtu=1400")
	assertDirective(t, dirs, "Options", "isolate=true")
}

func TestNetworks_DriverOpts_Unavailable(t *testing.T) {
	networks := types.Networks{
		"mynet": types.NetworkConfig{
			DriverOpts: types.Options{"mtu": "1400"},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 8}
	units := Networks(networks, cfg)
	dirs := units[0].Sections[0].Directives
	if hasDirective(dirs, "Options", "mtu=1400") {
		t.Fatal("should not emit Options before 6.0")
	}
	if !hasWarning(cfg, "mynet", "driver_opts") {
		t.Fatal("expected WarningSkipped for driver_opts at 5.8")
	}
}

func TestNetworks_IPAM(t *testing.T) {
	networks := types.Networks{
		"mynet": types.NetworkConfig{
			Ipam: types.IPAMConfig{
				Driver: "default",
				Config: []*types.IPAMPool{
					{Subnet: "10.0.0.0/24", Gateway: "10.0.0.1"},
					{Subnet: "10.0.1.0/24", Gateway: "10.0.1.1", IPRange: "10.0.1.0/28"},
				},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Networks(networks, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "IPAMDriver", "default")
	assertDirective(t, dirs, "Subnet", "10.0.0.0/24")
	assertDirective(t, dirs, "Gateway", "10.0.0.1")
	assertDirective(t, dirs, "Subnet", "10.0.1.0/24")
	assertDirective(t, dirs, "Gateway", "10.0.1.1")
	assertDirective(t, dirs, "IPRange", "10.0.1.0/28")
}

func TestNetworks_Internal_IPv6(t *testing.T) {
	networks := types.Networks{
		"mynet": types.NetworkConfig{
			Internal:   true,
			EnableIPv6: boolPtr(true),
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Networks(networks, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Internal", "true")
	assertDirective(t, dirs, "IPv6", "true")
}

func TestNetworks_IPv6_False(t *testing.T) {
	networks := types.Networks{
		"mynet": types.NetworkConfig{
			EnableIPv6: boolPtr(false),
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Networks(networks, cfg)
	dirs := units[0].Sections[0].Directives
	if hasDirective(dirs, "IPv6", "true") {
		t.Fatal("should not emit IPv6 when EnableIPv6 is false")
	}
}

func TestNetworks_Labels(t *testing.T) {
	networks := types.Networks{
		"mynet": types.NetworkConfig{
			Labels: types.Labels{"env": "prod"},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Networks(networks, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Label", "env=prod")
}

func TestNetworks_Labels_Unavailable(t *testing.T) {
	networks := types.Networks{
		"mynet": types.NetworkConfig{
			Labels: types.Labels{"env": "prod"},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 5}
	units := Networks(networks, cfg)
	dirs := units[0].Sections[0].Directives
	if hasDirective(dirs, "Label", "env=prod") {
		t.Fatal("should not emit Label before 5.6")
	}
	if !hasWarning(cfg, "mynet", "labels") {
		t.Fatal("expected WarningSkipped for labels at 5.5")
	}
}

func TestNetworks_External(t *testing.T) {
	networks := types.Networks{
		"hostnet": types.NetworkConfig{
			External: types.External(true),
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Networks(networks, cfg)
	if len(units) != 0 {
		t.Fatal("expected no units for external network")
	}
}

func TestNetworks_Empty(t *testing.T) {
	cfg := c2qtypes.DefaultConfig()
	units := Networks(types.Networks{}, cfg)
	if len(units) != 0 {
		t.Fatal("expected no units for empty networks")
	}
}
