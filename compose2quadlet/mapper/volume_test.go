package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestVolumes_Basic(t *testing.T) {
	volumes := types.Volumes{
		"data": types.VolumeConfig{
			Driver: "local",
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Volumes(volumes, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 volume unit, got %d", len(units))
	}
	u := units[0]
	if u.Type != c2qtypes.UnitVolume || u.Name != "data" {
		t.Fatalf("expected data volume unit, got %s/%s", u.Type, u.Name)
	}

	dirs := u.Sections[0].Directives
	if len(dirs) != 0 {
		t.Fatal("expected no directives for driver=local (default)")
	}
}

func TestVolumes_CustomDriver(t *testing.T) {
	volumes := types.Volumes{
		"data": types.VolumeConfig{Driver: "nfs"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Volumes(volumes, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Driver", "nfs")
}

func TestVolumes_Driver_Unavailable(t *testing.T) {
	volumes := types.Volumes{
		"data": types.VolumeConfig{Driver: "nfs"},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 4, Minor: 6}
	units := Volumes(volumes, cfg)
	dirs := units[0].Sections[0].Directives
	if hasDirective(dirs, "Driver", "nfs") {
		t.Fatal("should not emit Driver before 4.7")
	}
	if !hasWarning(cfg, "data", "driver") {
		t.Fatal("expected WarningSkipped for driver at 4.6")
	}
}

func TestVolumes_DriverOpts(t *testing.T) {
	volumes := types.Volumes{
		"data": types.VolumeConfig{DriverOpts: types.Options{"type": "nfs", "o": "addr=10.0.0.1"}},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Volumes(volumes, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Options", "type=nfs")
	assertDirective(t, dirs, "Options", "o=addr=10.0.0.1")
}

func TestVolumes_DriverOpts_Unavailable(t *testing.T) {
	volumes := types.Volumes{
		"data": types.VolumeConfig{DriverOpts: types.Options{"type": "nfs"}},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 8}
	units := Volumes(volumes, cfg)
	dirs := units[0].Sections[0].Directives
	if hasDirective(dirs, "Options", "type=nfs") {
		t.Fatal("should not emit Options before 6.0")
	}
	if !hasWarning(cfg, "data", "driver_opts") {
		t.Fatal("expected WarningSkipped for driver_opts at 5.8")
	}
}

func TestVolumes_Name(t *testing.T) {
	volumes := types.Volumes{
		"appdata": types.VolumeConfig{Name: "custom-vol-name"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Volumes(volumes, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "VolumeName", "custom-vol-name")
}

func TestVolumes_Labels(t *testing.T) {
	volumes := types.Volumes{
		"data": types.VolumeConfig{Labels: types.Labels{"backup": "daily"}},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Volumes(volumes, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Label", "backup=daily")
}

func TestVolumes_External(t *testing.T) {
	volumes := types.Volumes{
		"extvol": types.VolumeConfig{External: types.External(true)},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Volumes(volumes, cfg)
	if len(units) != 0 {
		t.Fatal("expected no units for external volume")
	}
}

func TestVolumes_Empty(t *testing.T) {
	cfg := c2qtypes.DefaultConfig()
	units := Volumes(types.Volumes{}, cfg)
	if len(units) != 0 {
		t.Fatal("expected no units for empty volumes")
	}
}
