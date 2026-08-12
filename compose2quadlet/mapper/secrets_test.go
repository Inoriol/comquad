package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestPremapSecrets_External(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Secrets: []types.ServiceSecretConfig{{Source: "db_password", Target: "/run/secrets/db_password"}},
	}
	secrets := types.Secrets{
		"db_password": {Name: "my-secret", External: types.External(true)},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "my-secret")
	if len(svc.Secrets) != 0 {
		t.Fatal("expected secrets to be stripped from service")
	}
}

func TestPremapSecrets_External_DefaultName(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Secrets: []types.ServiceSecretConfig{{Source: "db_password"}},
	}
	secrets := types.Secrets{
		"db_password": {External: types.External(true)},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password")
}

func TestPremapSecrets_File(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Secrets: []types.ServiceSecretConfig{{Source: "db_password"}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Volume", "/etc/secrets/db_password.txt:/run/secrets/db_password:ro")
}

func TestPremapSecrets_CustomTarget(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Secrets: []types.ServiceSecretConfig{{Source: "db_password", Target: "/custom/path/secret"}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Volume", "/etc/secrets/db_password.txt:/custom/path/secret:ro")
}

func TestPremapSecrets_Environment(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Secrets: []types.ServiceSecretConfig{{Source: "db_password"}},
	}
	secrets := types.Secrets{
		"db_password": {Environment: "DB_PASSWORD"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, secrets, nil, cfg)

	if len(dirs) != 0 {
		t.Fatal("expected no directives for environment secrets (not pre-resolved)")
	}
	if !hasWarning(cfg, "web", "secrets.db_password") {
		t.Fatal("expected WarningDegraded for environment secret")
	}
}

func TestPremapSecrets_Undefined(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Secrets: []types.ServiceSecretConfig{{Source: "nonexistent"}},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, types.Secrets{}, nil, cfg)

	if len(dirs) != 0 {
		t.Fatal("expected no directives for undefined secret")
	}
	if !hasWarning(cfg, "web", "secrets.nonexistent") {
		t.Fatal("expected WarningSkipped for undefined secret")
	}
}

func TestPremapSecrets_ExternalUnavailable(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Secrets: []types.ServiceSecretConfig{{Source: "s"}},
	}
	secrets := types.Secrets{
		"s": {External: types.External(true)},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 4, Minor: 4}
	dirs := PremapSecrets(&svc, secrets, nil, cfg)

	if hasDirective(dirs, "Secret", "") {
		t.Fatal("should not emit Secret before 4.5")
	}
	if !hasWarning(cfg, "web", "secrets.s") {
		t.Fatal("expected WarningSkipped for external secret at 4.4")
	}
}

func TestPremapConfigs_Basic(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Configs: []types.ServiceConfigObjConfig{{Source: "app_config", Target: "/etc/app/config.yml"}},
	}
	configs := types.Configs{
		"app_config": {File: "/etc/configs/app_config.yml"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, nil, configs, cfg)

	assertDirective(t, dirs, "Mount", "type=bind,source=/etc/configs/app_config.yml,destination=/etc/app/config.yml")
	if len(svc.Configs) != 0 {
		t.Fatal("expected configs to be stripped from service")
	}
}

func TestPremapConfigs_DefaultTarget(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Configs: []types.ServiceConfigObjConfig{{Source: "app_config"}},
	}
	configs := types.Configs{
		"app_config": {File: "/etc/configs/app_config.yml"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, nil, configs, cfg)

	assertDirective(t, dirs, "Mount", "type=bind,source=/etc/configs/app_config.yml,destination=/app_config")
}

func TestPremapConfigs_WithUIDGID(t *testing.T) {
	mode := types.FileMode(0440)
	svc := types.ServiceConfig{
		Name: "web",
		Configs: []types.ServiceConfigObjConfig{{
			Source: "app_config",
			UID:    "1000",
			GID:    "1000",
			Mode:   &mode,
		}},
	}
	configs := types.Configs{
		"app_config": {File: "/etc/configs/app_config.yml"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, nil, configs, cfg)

	assertDirective(t, dirs, "Mount", "type=bind,source=/etc/configs/app_config.yml,destination=/app_config,uid=1000,gid=1000,mode=0440")
}

func TestPremapConfigs_Undefined(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Configs: []types.ServiceConfigObjConfig{{Source: "nonexistent"}},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, nil, types.Configs{}, cfg)

	if len(dirs) != 0 {
		t.Fatal("expected no directives for undefined config")
	}
	if !hasWarning(cfg, "web", "configs.nonexistent") {
		t.Fatal("expected WarningSkipped for undefined config")
	}
}

func TestPremapSecrets_NoSecrets(t *testing.T) {
	svc := types.ServiceConfig{Name: "web"}
	cfg := c2qtypes.DefaultConfig()
	dirs := PremapSecrets(&svc, types.Secrets{}, types.Configs{}, cfg)

	if len(dirs) != 0 {
		t.Fatal("expected no directives when no secrets/configs")
	}
}
