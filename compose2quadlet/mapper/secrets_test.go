package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestPremapSecrets_External(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "db_password", Target: "/run/secrets/db_password"}}},
	}
	secrets := types.Secrets{
		"db_password": {Name: "my-secret", External: types.External(true)},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "my-secret,target=/run/secrets/db_password")
	if len(svc.Secrets) != 0 {
		t.Fatal("expected secrets to be stripped from service")
	}
}

func TestPremapSecrets_External_DefaultName(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "db_password"}}},
	}
	secrets := types.Secrets{
		"db_password": {External: types.External(true)},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password")
}

func TestPremapSecrets_File(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "db_password"}}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, units := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password.secretref")
	if len(units) != 1 {
		t.Fatalf("expected 1 secret unit, got %d", len(units))
	}
	if units[0].Type != c2qtypes.UnitCQSecret {
		t.Fatalf("expected cq-secret unit type, got %s", units[0].Type)
	}
	if units[0].Name != "db_password" {
		t.Fatalf("expected secret name db_password, got %s", units[0].Name)
	}
}

func TestPremapSecrets_CustomTarget(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "db_password", Target: "/custom/path/secret"}}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, units := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password.secretref,target=/custom/path/secret")
	if len(units) != 1 {
		t.Fatalf("expected 1 secret unit, got %d", len(units))
	}
}

func TestPremapSecrets_Environment(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "db_password"}}},
	}
	secrets := types.Secrets{
		"db_password": {Environment: "DB_PASSWORD"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, units := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password.secretref")
	if len(units) != 1 {
		t.Fatalf("expected 1 secret unit, got %d", len(units))
	}
	if units[0].Type != c2qtypes.UnitCQSecret {
		t.Fatalf("expected cq-secret unit type, got %s", units[0].Type)
	}
}

func TestPremapSecrets_Undefined(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "nonexistent"}}},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, types.Secrets{}, nil, cfg)

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
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "s"}}},
	}
	secrets := types.Secrets{
		"s": {External: types.External(true)},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 4, Minor: 4}
	dirs, _ := PremapSecrets(&svc, secrets, nil, cfg)

	if hasDirective(dirs, "Secret", "") {
		t.Fatal("should not emit Secret before 4.5")
	}
	if !hasWarning(cfg, "web", "secrets.s") {
		t.Fatal("expected WarningSkipped for external secret at 4.4")
	}
}

func TestPremapSecrets_NoSecrets(t *testing.T) {
	svc := types.ServiceConfig{Name: "web"}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, types.Secrets{}, types.Configs{}, cfg)

	if len(dirs) != 0 {
		t.Fatal("expected no directives when no secrets/configs")
	}
}

func TestPremapSecrets_CustomPermissions(t *testing.T) {
	mode := types.FileMode(0400)
	svc := types.ServiceConfig{
		Name: "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{
			Source: "db_password",
			UID:    "1000",
			GID:    "1000",
			Mode:   &mode,
		},
		}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, units := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password.secretref,uid=1000,gid=1000,mode=0400")
	if len(units) != 1 {
		t.Fatalf("expected 1 secret unit, got %d", len(units))
	}
}

func TestPremapSecrets_RelativeTarget(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "db_password", Target: "my_secret"}}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password.secretref,target=/run/secrets/my_secret")
}

func TestPremapSecrets_XTypeEnv(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{
			Source: "db_password",
			Extensions: types.Extensions{
				"x-type": "env",
		},
			},
		}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, units := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password.secretref,type=env")
	if len(units) != 1 {
		t.Fatalf("expected 1 secret unit, got %d", len(units))
	}
}

func TestPremapSecrets_XTypeEnvWithTarget(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{
			Source: "db_password",
			Extensions: types.Extensions{
				"x-type":   "env",
				"x-target": "DB_PASSWORD",
		},
			},
		}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password.secretref,type=env,target=DB_PASSWORD")
}

func TestPremapSecrets_XTargetOnly(t *testing.T) {
	svc := types.ServiceConfig{
		Name: "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{
			Source: "db_password",
			Extensions: types.Extensions{
				"x-target": "/custom/path",
		},
			},
		}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password.secretref,target=/custom/path")
}

func TestPremapSecrets_XTypeEnvWithPerms_Warning(t *testing.T) {
	mode := types.FileMode(0400)
	svc := types.ServiceConfig{
		Name: "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{
			Source: "db_password",
			UID:    "1000",
			Mode:   &mode,
			Extensions: types.Extensions{
				"x-type": "env",
		},
			},
		}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "db_password.secretref")
	if !hasWarning(cfg, "web", "secrets.db_password") {
		t.Fatal("expected warning for x-type: env with uid/mode")
	}
}

func TestPremapSecrets_VersionFallback_File(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "db_password"}}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 4, Minor: 4}
	dirs, units := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Volume", "/etc/secrets/db_password.txt:/run/secrets/db_password:ro")
	if len(units) != 0 {
		t.Fatalf("expected no secret units for version fallback, got %d", len(units))
	}
}

func TestPremapSecrets_VersionFallback_CustomPerms(t *testing.T) {
	mode := types.FileMode(0400)
	svc := types.ServiceConfig{
		Name: "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{
			Source: "db_password",
			UID:    "1000",
			Mode:   &mode,
		},
		}},
	}
	secrets := types.Secrets{
		"db_password": {File: "/etc/secrets/db_password.txt"},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 4, Minor: 4}
	dirs, units := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Volume", "/etc/secrets/db_password.txt:/run/secrets/db_password:ro")
	if len(units) != 0 {
		t.Fatalf("expected no secret units for version fallback, got %d", len(units))
	}
}

func TestPremapSecrets_ExternalWithOptions(t *testing.T) {
	mode := types.FileMode(0400)
	svc := types.ServiceConfig{
		Name: "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{
			Source: "db_password",
			Target: "/custom/path",
			UID:    "1000",
			Mode:   &mode,
		},
		}},
	}
	secrets := types.Secrets{
		"db_password": {Name: "my-secret", External: types.External(true)},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "my-secret,target=/custom/path,uid=1000,mode=0400")
}

func TestPremapConfigs_Basic(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Configs: []types.ServiceConfigObjConfig{{Source: "app_config", Target: "/etc/app/config.yml"}},
		},
	}
	configs := types.Configs{
		"app_config": {File: "/etc/configs/app_config.yml"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, nil, configs, cfg)

	assertDirective(t, dirs, "Mount", "type=bind,source=/etc/configs/app_config.yml,destination=/etc/app/config.yml")
	if len(svc.Configs) != 0 {
		t.Fatal("expected configs to be stripped from service")
	}
}

func TestPremapConfigs_DefaultTarget(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Configs: []types.ServiceConfigObjConfig{{Source: "app_config"}},
		},
	}
	configs := types.Configs{
		"app_config": {File: "/etc/configs/app_config.yml"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, nil, configs, cfg)

	assertDirective(t, dirs, "Mount", "type=bind,source=/etc/configs/app_config.yml,destination=/app_config")
}

func TestPremapConfigs_WithUIDGID(t *testing.T) {
	mode := types.FileMode(0440)
	svc := types.ServiceConfig{
		Name: "web",
		ContainerSpec: types.ContainerSpec{Configs: []types.ServiceConfigObjConfig{{
			Source: "app_config",
			UID:    "1000",
			GID:    "1000",
			Mode:   &mode,
		}}},
	}
	configs := types.Configs{
		"app_config": {File: "/etc/configs/app_config.yml"},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, nil, configs, cfg)

	assertDirective(t, dirs, "Mount", "type=bind,source=/etc/configs/app_config.yml,destination=/app_config,uid=1000,gid=1000,mode=0440")
}

func TestPremapSecrets_Driver(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "encrypted_secret"}}},
	}
	secrets := types.Secrets{
		"encrypted_secret": {
			File:   "/etc/secrets/secret.txt.gpg",
			Driver: "pass",
		},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, units := PremapSecrets(&svc, secrets, nil, cfg)

	assertDirective(t, dirs, "Secret", "encrypted_secret.secretref")
	if len(units) != 1 {
		t.Fatalf("expected 1 secret unit, got %d", len(units))
	}
	unit := units[0]
	if unit.Type != c2qtypes.UnitCQSecret {
		t.Fatalf("expected cq-secret unit type, got %s", unit.Type)
	}
	var secretSection *c2qtypes.Section
	for i := range unit.Sections {
		if unit.Sections[i].Name == c2qtypes.SectionSecret {
			secretSection = &unit.Sections[i]
			break
		}
	}
	if secretSection == nil {
		t.Fatal("expected [Secret] section")
	}
	assertDirective(t, secretSection.Directives, "Driver", "pass")
}

func TestPremapSecrets_DriverOpts(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "shell_secret"}}},
	}
	secrets := types.Secrets{
		"shell_secret": {
			Driver: "shell",
			DriverOpts: map[string]string{
				"list":   "/usr/local/bin/list.sh",
				"lookup": "/usr/local/bin/lookup.sh",
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	_, units := PremapSecrets(&svc, secrets, nil, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 secret unit, got %d", len(units))
	}
	unit := units[0]
	var secretSection *c2qtypes.Section
	for i := range unit.Sections {
		if unit.Sections[i].Name == c2qtypes.SectionSecret {
			secretSection = &unit.Sections[i]
			break
		}
	}
	if secretSection == nil {
		t.Fatal("expected [Secret] section")
	}
	assertDirective(t, secretSection.Directives, "Source", "driver")
	assertDirective(t, secretSection.Directives, "Driver", "shell")
	assertDirective(t, secretSection.Directives, "DriverOpts", "list=/usr/local/bin/list.sh,lookup=/usr/local/bin/lookup.sh")
}

func TestPremapSecrets_DriverWithFile(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Secrets: []types.ServiceSecretConfig{{Source: "encrypted_secret"}}},
	}
	secrets := types.Secrets{
		"encrypted_secret": {
			File:   "/etc/secrets/secret.txt.gpg",
			Driver: "pass",
		},
	}
	cfg := c2qtypes.DefaultConfig()
	_, units := PremapSecrets(&svc, secrets, nil, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 secret unit, got %d", len(units))
	}
	unit := units[0]
	var secretSection *c2qtypes.Section
	for i := range unit.Sections {
		if unit.Sections[i].Name == c2qtypes.SectionSecret {
			secretSection = &unit.Sections[i]
			break
		}
	}
	if secretSection == nil {
		t.Fatal("expected [Secret] section")
	}
	assertDirective(t, secretSection.Directives, "Source", "file")
	assertDirective(t, secretSection.Directives, "Path", "/etc/secrets/secret.txt.gpg")
	assertDirective(t, secretSection.Directives, "Driver", "pass")
}

func TestPremapConfigs_Undefined(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		ContainerSpec: types.ContainerSpec{Configs: []types.ServiceConfigObjConfig{{Source: "nonexistent"}},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs, _ := PremapSecrets(&svc, nil, types.Configs{}, cfg)

	if len(dirs) != 0 {
		t.Fatal("expected no directives for undefined config")
	}
	if !hasWarning(cfg, "web", "configs.nonexistent") {
		t.Fatal("expected WarningSkipped for undefined config")
	}
}
