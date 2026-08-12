package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestImages_Basic(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "nginx:latest"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Images(services, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 image unit, got %d", len(units))
	}
	u := units[0]
	if u.Type != c2qtypes.UnitImage || u.Name != "web" {
		t.Fatalf("expected web image unit, got %s/%s", u.Type, u.Name)
	}

	dirs := u.Sections[0].Directives
	assertDirective(t, dirs, "Image", "docker.io/library/nginx:latest")
}

func TestImages_NoImage(t *testing.T) {
	services := types.Services{
		"nosvc": types.ServiceConfig{Name: "nosvc"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Images(services, cfg)
	if len(units) != 0 {
		t.Fatal("expected no image units for service without image")
	}
}

func TestImages_PullPolicy(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "nginx:latest", PullPolicy: "always"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Images(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Policy", "always")
}

func TestImages_PullPolicy_Unavailable(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "nginx:latest", PullPolicy: "always"},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 5}
	units := Images(services, cfg)
	dirs := units[0].Sections[0].Directives
	if hasDirective(dirs, "Policy", "always") {
		t.Fatal("should not emit Policy before 5.6")
	}
	if !hasWarning(cfg, "web", "pull_policy") {
		t.Fatal("expected WarningSkipped for pull_policy at 5.5")
	}
}

func TestImages_IfNotPresent(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "nginx:latest", PullPolicy: "if_not_present"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Images(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Policy", "missing")
}

func TestImages_InvalidPolicy(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "nginx:latest", PullPolicy: "daily"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Images(services, cfg)
	dirs := units[0].Sections[0].Directives
	if hasDirective(dirs, "Policy", "") {
		t.Fatal("should not emit Policy for invalid pull_policy")
	}
}

func TestImages_Platform(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "alpine:latest", Platform: "linux/arm64/v8"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Images(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "OS", "linux")
	assertDirective(t, dirs, "Arch", "arm64")
	assertDirective(t, dirs, "Variant", "v8")
}

func TestImages_Platform_OSOnly(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "alpine:latest", Platform: "linux"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Images(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "OS", "linux")
	if hasDirective(dirs, "Arch", "") {
		t.Fatal("should not emit Arch when not specified")
	}
}

func TestImages_Retry(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "nginx:latest"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Images(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Retry", "3")
	assertDirective(t, dirs, "RetryDelay", "5s")
}

func TestImages_Retry_Unavailable(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "nginx:latest"},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 4}
	units := Images(services, cfg)
	dirs := units[0].Sections[0].Directives
	if hasDirective(dirs, "Retry", "3") {
		t.Fatal("should not emit Retry before 5.5")
	}
}

func TestImages_Multiple(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "nginx:latest"},
		"db":  types.ServiceConfig{Name: "db", Image: "postgres:15"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Images(services, cfg)

	if len(units) != 2 {
		t.Fatalf("expected 2 image units, got %d", len(units))
	}
	names := map[string]bool{}
	for _, u := range units {
		names[u.Name] = true
	}
	if !names["web"] || !names["db"] {
		t.Fatal("expected web and db image units")
	}
}
