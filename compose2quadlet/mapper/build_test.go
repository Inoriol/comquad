package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestBuilds_Basic(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Build: &types.BuildConfig{Context: "/app", Dockerfile: "Dockerfile"}},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 build unit, got %d", len(units))
	}
	u := units[0]
	if u.Type != c2qtypes.UnitBuild || u.Name != "web" {
		t.Fatalf("expected web build unit, got %s/%s", u.Type, u.Name)
	}

	dirs := u.Sections[0].Directives
	assertDirective(t, dirs, "SetWorkingDirectory", "/app")
	assertDirective(t, dirs, "File", "Dockerfile")
	assertDirective(t, dirs, "ImageTag", "web:latest")
}

func TestBuilds_NoBuild(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Image: "nginx:latest"},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)
	if len(units) != 0 {
		t.Fatal("expected no build units for service without build")
	}
}

func TestBuilds_VersionGate(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Build: &types.BuildConfig{Context: "/app"}},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 1}
	units := Builds(services, cfg)
	if len(units) != 0 {
		t.Fatal("expected no build units before 5.2")
	}
}

func TestBuilds_DockerfileInline(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Build: &types.BuildConfig{DockerfileInline: "FROM scratch", Context: "."}},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "File", "FROM scratch")
}

func TestBuilds_Target(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Build: &types.BuildConfig{Context: "/app", Target: "builder"}},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Target", "builder")
}

func TestBuilds_Network(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Build: &types.BuildConfig{Context: "/app", Network: "host"}},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Network", "host")
}

func TestBuilds_NoCache(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Build: &types.BuildConfig{Context: "/app", NoCache: true}},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "PodmanArgs", "--no-cache")
}

func TestBuilds_Labels(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Build: &types.BuildConfig{Context: "/app", Labels: types.Labels{"version": "1.0"}}},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Label", "version=1.0")
}

func TestBuilds_Tags(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{Name: "web", Build: &types.BuildConfig{Context: "/app", Tags: types.StringList{"myapp:latest", "myapp:v1"}}},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "ImageTag", "myapp:latest")
	assertDirective(t, dirs, "ImageTag", "myapp:v1")
}

func TestBuilds_Secrets(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{
			Name:  "web",
			Build: &types.BuildConfig{Context: "/app", Secrets: []types.ServiceSecretConfig{{Source: "mysecret"}}},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Secret", "mysecret")
}

func TestBuilds_Args(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{
			Name:  "web",
			Build: &types.BuildConfig{Context: "/app", Args: types.MappingWithEquals{"NODE_ENV": strPtr("production"), "VERSION": nil}},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Builds(services, cfg)
	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "BuildArg", "NODE_ENV=production")
	assertDirective(t, dirs, "BuildArg", "VERSION")
}

func TestBuilds_Args_Unavailable(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{
			Name:  "web",
			Build: &types.BuildConfig{Context: "/app", Args: types.MappingWithEquals{"NODE_ENV": strPtr("production")}},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 6}
	units := Builds(services, cfg)
	dirs := units[0].Sections[0].Directives
	if hasDirective(dirs, "BuildArg", "NODE_ENV=production") {
		t.Fatal("should not emit BuildArg before 5.7")
	}
	if !hasWarning(cfg, "web", "build.args") {
		t.Fatal("expected WarningSkipped for build.args at 5.6")
	}
}
