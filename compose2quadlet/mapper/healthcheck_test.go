package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestHealthcheck_CMD(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", HealthCheck: &types.HealthCheckConfig{Test: types.HealthCheckTest{"CMD", "curl", "localhost"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Healthcheck(svc, cfg)
	assertDirective(t, dirs, "HealthCmd", "curl localhost")
}

func TestHealthcheck_CMDSHELL(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", HealthCheck: &types.HealthCheckConfig{Test: types.HealthCheckTest{"CMD-SHELL", "curl -f http://localhost || exit 1"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Healthcheck(svc, cfg)
	assertDirective(t, dirs, "HealthCmd", "/bin/sh -c curl -f http://localhost || exit 1")
}

func TestHealthcheck_NONE(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", HealthCheck: &types.HealthCheckConfig{Test: types.HealthCheckTest{"NONE"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Healthcheck(svc, cfg)
	if len(dirs) > 0 {
		t.Fatalf("expected no directives for NONE test, got %v", dirs)
	}
}

func TestHealthcheck_Disable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", HealthCheck: &types.HealthCheckConfig{Disable: true, Test: types.HealthCheckTest{"CMD", "curl", "localhost"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Healthcheck(svc, cfg)
	if len(dirs) > 0 {
		t.Fatalf("expected no directives when disabled, got %v", dirs)
	}
}

func TestHealthcheck_Interval(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", HealthCheck: &types.HealthCheckConfig{Test: types.HealthCheckTest{"CMD", "true"}, Interval: durationPtr(types.Duration(10_000_000_000)), Timeout: durationPtr(types.Duration(5_000_000_000)), Retries: uintPtr(3)}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Healthcheck(svc, cfg)
	assertDirective(t, dirs, "HealthInterval", "10s")
	assertDirective(t, dirs, "HealthTimeout", "5s")
	assertDirective(t, dirs, "HealthRetries", "3")
}

func TestHealthcheck_StartPeriod(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", HealthCheck: &types.HealthCheckConfig{Test: types.HealthCheckTest{"CMD", "true"}, StartPeriod: durationPtr(types.Duration(30_000_000_000)), StartInterval: durationPtr(types.Duration(2_000_000_000))}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Healthcheck(svc, cfg)
	assertDirective(t, dirs, "HealthStartPeriod", "30s")
	assertDirective(t, dirs, "HealthStartupInterval", "2s")
}

func TestHealthcheck_Nil(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", HealthCheck: nil}
	cfg := c2qtypes.DefaultConfig()
	dirs := Healthcheck(svc, cfg)
	if len(dirs) > 0 {
		t.Fatalf("expected no directives for nil healthcheck, got %v", dirs)
	}
}
