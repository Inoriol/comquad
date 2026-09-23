package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestExtractContainerExtensions(t *testing.T) {
	tests := []struct {
		name     string
		svc      types.ServiceConfig
		wantKey  string
		wantVal  string
		wantNone bool
	}{
		{
			name: "timezone extension",
			svc: types.ServiceConfig{
				Name: "web",
				Extensions: types.Extensions{
					"x-container": map[string]interface{}{
						"Timezone": "Europe/Berlin",
					},
				},
			},
			wantKey: "Timezone",
			wantVal: "Europe/Berlin",
		},
		{
			name: "notify extension",
			svc: types.ServiceConfig{
				Name: "web",
				Extensions: types.Extensions{
					"x-container": map[string]interface{}{
						"Notify": "healthy",
					},
				},
			},
			wantKey: "Notify",
			wantVal: "healthy",
		},
		{
			name: "boolean extension",
			svc: types.ServiceConfig{
				Name: "web",
				Extensions: types.Extensions{
					"x-container": map[string]interface{}{
						"EnvironmentHost": true,
					},
				},
			},
			wantKey: "EnvironmentHost",
			wantVal: "true",
		},
		{
			name: "no extension",
			svc: types.ServiceConfig{
				Name: "web",
			},
			wantNone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := c2qtypes.DefaultConfig()
			dirs := ExtractContainerExtensions(tt.svc, cfg)

			if tt.wantNone {
				if len(dirs) != 0 {
					t.Fatalf("expected no directives, got %v", dirs)
				}
				return
			}

			found := false
			for _, d := range dirs {
				if d.Key == tt.wantKey {
					if len(d.Values) > 0 && d.Values[0] == tt.wantVal {
						found = true
						break
					}
				}
			}
			if !found {
				t.Fatalf("expected directive %s=%s, got %v", tt.wantKey, tt.wantVal, dirs)
			}
		})
	}
}

func TestExtractImageExtensions(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		Extensions: types.Extensions{
			"x-image": map[string]interface{}{
				"AllTags":  true,
				"AuthFile": "/path/to/auth.json",
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	dirs := ExtractImageExtensions(svc, cfg)

	if len(dirs) != 2 {
		t.Fatalf("expected 2 directives, got %d", len(dirs))
	}

	assertDirective(t, dirs, "AllTags", "true")
	assertDirective(t, dirs, "AuthFile", "/path/to/auth.json")
}

func TestExtractBuildExtensions(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		Extensions: types.Extensions{
			"x-build": map[string]interface{}{
				"DNS":         "8.8.8.8",
				"Environment": []interface{}{"BUILD_ENV=production"},
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	dirs := ExtractBuildExtensions(svc, cfg)

	if len(dirs) != 2 {
		t.Fatalf("expected 2 directives, got %d", len(dirs))
	}

	assertDirective(t, dirs, "DNS", "8.8.8.8")
	assertDirective(t, dirs, "Environment", "BUILD_ENV=production")
}

func TestExtractSystemdExtensions(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		Extensions: types.Extensions{
			"x-systemd": map[string]interface{}{
				"Service": map[string]interface{}{
					"MemoryMax": "512M",
					"Restart":   "on-failure",
				},
				"Unit": map[string]interface{}{
					"After":         "network-online.target",
					"Documentation": "https://example.com",
				},
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	svcDirs, unitDirs := ExtractSystemdExtensions(svc, cfg)

	if len(svcDirs) != 2 {
		t.Fatalf("expected 2 service directives, got %d", len(svcDirs))
	}
	if len(unitDirs) != 2 {
		t.Fatalf("expected 2 unit directives, got %d", len(unitDirs))
	}

	assertDirective(t, svcDirs, "MemoryMax", "512M")
	assertDirective(t, svcDirs, "Restart", "on-failure")
	assertDirective(t, unitDirs, "After", "network-online.target")
	assertDirective(t, unitDirs, "Documentation", "https://example.com")
}

func TestExtractNetworkExtensions(t *testing.T) {
	nc := types.NetworkConfig{
		Extensions: types.Extensions{
			"x-network": map[string]interface{}{
				"ContainersConfModule": "/etc/containers/my.conf",
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	dirs := ExtractNetworkExtensions("frontend", nc, cfg)

	if len(dirs) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(dirs))
	}

	assertDirective(t, dirs, "ContainersConfModule", "/etc/containers/my.conf")
}

func TestExtractVolumeExtensions(t *testing.T) {
	vc := types.VolumeConfig{
		Extensions: types.Extensions{
			"x-volume": map[string]interface{}{
				"User":  "1000",
				"Group": "1000",
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	dirs := ExtractVolumeExtensions("data", vc, cfg)

	if len(dirs) != 2 {
		t.Fatalf("expected 2 directives, got %d", len(dirs))
	}

	assertDirective(t, dirs, "User", "1000")
	assertDirective(t, dirs, "Group", "1000")
}

func TestMergeDirectives(t *testing.T) {
	base := []c2qtypes.Directive{
		{Key: "Image", Values: []string{"nginx"}},
		{Key: "Environment", Values: []string{"FOO=bar"}},
	}

	override := []c2qtypes.Directive{
		{Key: "Image", Values: []string{"redis"}},
		{Key: "Timezone", Values: []string{"UTC"}},
	}

	result := MergeDirectives(base, override)

	if len(result) != 3 {
		t.Fatalf("expected 3 directives, got %d", len(result))
	}

	assertDirective(t, result, "Image", "redis")
	assertDirective(t, result, "Environment", "FOO=bar")
	assertDirective(t, result, "Timezone", "UTC")
}

func TestMergeDirectives_EmptyBase(t *testing.T) {
	override := []c2qtypes.Directive{
		{Key: "Image", Values: []string{"nginx"}},
	}

	result := MergeDirectives(nil, override)

	if len(result) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(result))
	}

	assertDirective(t, result, "Image", "nginx")
}

func TestMergeDirectives_EmptyOverride(t *testing.T) {
	base := []c2qtypes.Directive{
		{Key: "Image", Values: []string{"nginx"}},
	}

	result := MergeDirectives(base, nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(result))
	}

	assertDirective(t, result, "Image", "nginx")
}

func TestUnknownDirectiveWarning(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		Extensions: types.Extensions{
			"x-container": map[string]interface{}{
				"UnknownDirective": "value",
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	dirs := ExtractContainerExtensions(svc, cfg)

	if len(dirs) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(dirs))
	}

	if len(cfg.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(cfg.Warnings))
	}

	if cfg.Warnings[0].Level != c2qtypes.WarningSkipped {
		t.Fatalf("expected WarningSkipped, got %d", cfg.Warnings[0].Level)
	}
}

func TestInvalidExtensionFormat(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		Extensions: types.Extensions{
			"x-container": "not a map",
		},
	}

	cfg := c2qtypes.DefaultConfig()
	dirs := ExtractContainerExtensions(svc, cfg)

	if len(dirs) != 0 {
		t.Fatalf("expected 0 directives, got %d", len(dirs))
	}

	if len(cfg.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(cfg.Warnings))
	}
}

func TestExtractExtensions_NumericValues(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		Extensions: types.Extensions{
			"x-container": map[string]interface{}{
				"PidsLimit":  100,
				"ShmSize":    67108864,
				"FloatValue": 3.14,
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	dirs := ExtractContainerExtensions(svc, cfg)

	if len(dirs) != 3 {
		t.Fatalf("expected 3 directives, got %d", len(dirs))
	}

	assertDirective(t, dirs, "PidsLimit", "100")
	assertDirective(t, dirs, "ShmSize", "67108864")
	assertDirective(t, dirs, "FloatValue", "3.14")
}

func TestExtractSystemdExtensions_PartialService(t *testing.T) {
	tests := []struct {
		name         string
		extensions   types.Extensions
		wantSvcCount int
		wantUnitCount int
	}{
		{
			name: "only service",
			extensions: types.Extensions{
				"x-systemd": map[string]interface{}{
					"Service": map[string]interface{}{
						"MemoryMax": "512M",
					},
				},
			},
			wantSvcCount:  1,
			wantUnitCount: 0,
		},
		{
			name: "only unit",
			extensions: types.Extensions{
				"x-systemd": map[string]interface{}{
					"Unit": map[string]interface{}{
						"After": "network.target",
					},
				},
			},
			wantSvcCount:  0,
			wantUnitCount: 1,
		},
		{
			name: "both service and unit",
			extensions: types.Extensions{
				"x-systemd": map[string]interface{}{
					"Service": map[string]interface{}{
						"MemoryMax": "512M",
					},
					"Unit": map[string]interface{}{
						"After": "network.target",
					},
				},
			},
			wantSvcCount:  1,
			wantUnitCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := types.ServiceConfig{
				Name:       "web",
				Extensions: tt.extensions,
			}
			cfg := c2qtypes.DefaultConfig()
			svcDirs, unitDirs := ExtractSystemdExtensions(svc, cfg)

			if len(svcDirs) != tt.wantSvcCount {
				t.Fatalf("expected %d service directives, got %d", tt.wantSvcCount, len(svcDirs))
			}
			if len(unitDirs) != tt.wantUnitCount {
				t.Fatalf("expected %d unit directives, got %d", tt.wantUnitCount, len(unitDirs))
			}
		})
	}
}

func TestExtractExtensions_InvalidSystemdFormat(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		Extensions: types.Extensions{
			"x-systemd": "not a map",
		},
	}

	cfg := c2qtypes.DefaultConfig()
	svcDirs, unitDirs := ExtractSystemdExtensions(svc, cfg)

	if len(svcDirs) != 0 {
		t.Fatalf("expected 0 service directives, got %d", len(svcDirs))
	}
	if len(unitDirs) != 0 {
		t.Fatalf("expected 0 unit directives, got %d", len(unitDirs))
	}
	if len(cfg.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(cfg.Warnings))
	}
}

func TestExtractExtensions_InvalidSectionFormat(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		Extensions: types.Extensions{
			"x-systemd": map[string]interface{}{
				"Service": "not a map",
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	svcDirs, unitDirs := ExtractSystemdExtensions(svc, cfg)

	if len(svcDirs) != 0 {
		t.Fatalf("expected 0 service directives, got %d", len(svcDirs))
	}
	if len(unitDirs) != 0 {
		t.Fatalf("expected 0 unit directives, got %d", len(unitDirs))
	}
	if len(cfg.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(cfg.Warnings))
	}
}

func TestContainerExtensions_OverrideBehavior(t *testing.T) {
	svc := types.ServiceConfig{
		Name:        "web",
		PidsLimit:   100,
		Extensions: types.Extensions{
			"x-container": map[string]interface{}{
				"PidsLimit": 200,
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)

	var pidsLimitValues []string
	for _, d := range dirs {
		if d.Key == "PidsLimit" {
			pidsLimitValues = append(pidsLimitValues, d.Values...)
		}
	}

	if len(pidsLimitValues) != 1 {
		t.Fatalf("expected exactly 1 PidsLimit directive, got %d: %v", len(pidsLimitValues), pidsLimitValues)
	}
	if pidsLimitValues[0] != "200" {
		t.Fatalf("expected PidsLimit=200 (from x-extension), got %s", pidsLimitValues[0])
	}
}

func TestServiceExtensions_OverrideBehavior(t *testing.T) {
	svc := types.ServiceConfig{
		Name:      "web",
		MemLimit:  536870912,
		Extensions: types.Extensions{
			"x-systemd": map[string]interface{}{
				"Service": map[string]interface{}{
					"MemoryMax": "1G",
				},
			},
		},
	}

	cfg := c2qtypes.DefaultConfig()
	dirs := Service(svc, cfg)

	var memoryMaxValues []string
	for _, d := range dirs {
		if d.Key == "MemoryMax" {
			memoryMaxValues = append(memoryMaxValues, d.Values...)
		}
	}

	if len(memoryMaxValues) != 1 {
		t.Fatalf("expected exactly 1 MemoryMax directive, got %d: %v", len(memoryMaxValues), memoryMaxValues)
	}
	if memoryMaxValues[0] != "1G" {
		t.Fatalf("expected MemoryMax=1G (from x-extension), got %s", memoryMaxValues[0])
	}
}
