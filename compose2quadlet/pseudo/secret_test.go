package pseudo

import (
	"testing"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestConvertSecret_Environment(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQSecret,
		Name: "db_password",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionSecret,
				Directives: []c2qtypes.Directive{
					{Key: "Source", Values: []string{"environment"}},
					{Key: "Environment", Values: []string{"DB_PASSWORD"}},
				},
			},
		},
	}

	result := convertSecret(unit, "/run/comquad/secrets")

	if result.Type != c2qtypes.UnitCQSecret {
		t.Fatalf("expected cq-secret type, got %s", result.Type)
	}
	if result.Name != "db_password" {
		t.Fatalf("expected name db_password, got %s", result.Name)
	}

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	if serviceSection == nil {
		t.Fatal("expected [Service] section")
	}

	hasExecStart := false
	hasExecStop := false
	hasEnvMarker := false
	for _, d := range serviceSection.Directives {
		if d.Key == "ExecStart" && len(d.Values) > 0 {
			hasExecStart = true
			expected := "/bin/sh -c '/usr/bin/podman secret create --replace db_password /run/comquad/secrets/db_password.env'"
			if d.Values[0] != expected {
				t.Errorf("expected ExecStart=%s, got %s", expected, d.Values[0])
			}
		}
		if d.Key == "ExecStop" && len(d.Values) > 0 {
			hasExecStop = true
			expected := "-/usr/bin/podman secret rm db_password"
			if d.Values[0] != expected {
				t.Errorf("expected ExecStop=%s, got %s", expected, d.Values[0])
			}
		}
		if d.Key == "Environment" && len(d.Values) > 0 {
			hasEnvMarker = true
			expected := "COMQUAD_SECRET_ENV=DB_PASSWORD"
			if d.Values[0] != expected {
				t.Errorf("expected Environment=%s, got %s", expected, d.Values[0])
			}
		}
	}

	if !hasExecStart {
		t.Error("expected ExecStart directive")
	}
	if !hasExecStop {
		t.Error("expected ExecStop directive")
	}
	if !hasEnvMarker {
		t.Error("expected COMQUAD_SECRET_ENV marker")
	}
}

func TestConvertSecret_File(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQSecret,
		Name: "api_key",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionSecret,
				Directives: []c2qtypes.Directive{
					{Key: "Source", Values: []string{"file"}},
					{Key: "Path", Values: []string{"/etc/secrets/api.key"}},
				},
			},
		},
	}

	result := convertSecret(unit, "/run/comquad/secrets")

	if result.Type != c2qtypes.UnitCQSecret {
		t.Fatalf("expected cq-secret type, got %s", result.Type)
	}

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	if serviceSection == nil {
		t.Fatal("expected [Service] section")
	}

	hasExecStart := false
	hasExecStop := false
	for _, d := range serviceSection.Directives {
		if d.Key == "ExecStart" && len(d.Values) > 0 {
			hasExecStart = true
			expected := "/usr/bin/podman secret create --replace api_key /etc/secrets/api.key"
			if d.Values[0] != expected {
				t.Errorf("expected ExecStart=%s, got %s", expected, d.Values[0])
			}
		}
		if d.Key == "ExecStop" && len(d.Values) > 0 {
			hasExecStop = true
			expected := "-/usr/bin/podman secret rm api_key"
			if d.Values[0] != expected {
				t.Errorf("expected ExecStop=%s, got %s", expected, d.Values[0])
			}
		}
	}

	if !hasExecStart {
		t.Error("expected ExecStart directive")
	}
	if !hasExecStop {
		t.Error("expected ExecStop directive")
	}
}

func TestConvertSecret_WithDriver(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQSecret,
		Name: "encrypted_secret",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionSecret,
				Directives: []c2qtypes.Directive{
					{Key: "Source", Values: []string{"file"}},
					{Key: "Path", Values: []string{"/etc/secrets/secret.txt.gpg"}},
					{Key: "Driver", Values: []string{"pass"}},
				},
			},
		},
	}

	result := convertSecret(unit, "/run/comquad/secrets")

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	if serviceSection == nil {
		t.Fatal("expected [Service] section")
	}

	for _, d := range serviceSection.Directives {
		if d.Key == "ExecStart" && len(d.Values) > 0 {
			expected := "/usr/bin/podman secret create --driver=pass --replace encrypted_secret /etc/secrets/secret.txt.gpg"
			if d.Values[0] != expected {
				t.Errorf("expected ExecStart=%s, got %s", expected, d.Values[0])
			}
			return
		}
	}
	t.Error("expected ExecStart directive")
}

func TestConvertSecret_WithDriverOpts(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQSecret,
		Name: "shell_secret",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionSecret,
				Directives: []c2qtypes.Directive{
					{Key: "Source", Values: []string{"file"}},
					{Key: "Path", Values: []string{"/dev/null"}},
					{Key: "Driver", Values: []string{"shell"}},
					{Key: "DriverOpts", Values: []string{"list=/bin/list.sh,lookup=/bin/lookup.sh"}},
				},
			},
		},
	}

	result := convertSecret(unit, "/run/comquad/secrets")

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	if serviceSection == nil {
		t.Fatal("expected [Service] section")
	}

	for _, d := range serviceSection.Directives {
		if d.Key == "ExecStart" && len(d.Values) > 0 {
			expected := "/usr/bin/podman secret create --driver=shell --driver-opts=list=/bin/list.sh,lookup=/bin/lookup.sh --replace shell_secret /dev/null"
			if d.Values[0] != expected {
				t.Errorf("expected ExecStart=%s, got %s", expected, d.Values[0])
			}
			return
		}
	}
	t.Error("expected ExecStart directive")
}

func TestConvertSecret_EnvironmentWithDriver(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQSecret,
		Name: "env_secret",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionSecret,
				Directives: []c2qtypes.Directive{
					{Key: "Source", Values: []string{"environment"}},
					{Key: "Environment", Values: []string{"MY_SECRET"}},
					{Key: "Driver", Values: []string{"pass"}},
				},
			},
		},
	}

	result := convertSecret(unit, "/run/comquad/secrets")

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	if serviceSection == nil {
		t.Fatal("expected [Service] section")
	}

	for _, d := range serviceSection.Directives {
		if d.Key == "ExecStart" && len(d.Values) > 0 {
			expected := "/bin/sh -c '/usr/bin/podman secret create --driver=pass --replace env_secret /run/comquad/secrets/env_secret.env'"
			if d.Values[0] != expected {
				t.Errorf("expected ExecStart=%s, got %s", expected, d.Values[0])
			}
			return
		}
	}
	t.Error("expected ExecStart directive")
}

func TestConvertSecret_DriverOnly(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQSecret,
		Name: "shell_secret",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionSecret,
				Directives: []c2qtypes.Directive{
					{Key: "Source", Values: []string{"driver"}},
					{Key: "Driver", Values: []string{"shell"}},
					{Key: "DriverOpts", Values: []string{"list=/bin/list.sh,lookup=/bin/lookup.sh"}},
				},
			},
		},
	}

	result := convertSecret(unit, "/run/comquad/secrets")

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	if serviceSection == nil {
		t.Fatal("expected [Service] section")
	}

	for _, d := range serviceSection.Directives {
		if d.Key == "ExecStart" && len(d.Values) > 0 {
			expected := "/bin/sh -c '/usr/bin/podman secret create --driver=shell --driver-opts=list=/bin/list.sh,lookup=/bin/lookup.sh --replace shell_secret -'"
			if d.Values[0] != expected {
				t.Errorf("expected ExecStart=%s, got %s", expected, d.Values[0])
			}
			return
		}
	}
	t.Error("expected ExecStart directive")
}

func TestConvertPseudoUnits(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		{
			Type: c2qtypes.UnitContainer,
			Name: "web",
			Sections: []c2qtypes.Section{
				{Name: c2qtypes.SectionContainer},
			},
		},
		{
			Type: c2qtypes.UnitCQSecret,
			Name: "db_password",
			Sections: []c2qtypes.Section{
				{
					Name: c2qtypes.SectionSecret,
					Directives: []c2qtypes.Directive{
						{Key: "Source", Values: []string{"environment"}},
						{Key: "Environment", Values: []string{"DB_PASSWORD"}},
					},
				},
			},
		},
		{
			Type: c2qtypes.UnitNetwork,
			Name: "frontend",
			Sections: []c2qtypes.Section{
				{Name: c2qtypes.SectionNetwork},
			},
		},
	}

	result := ConvertPseudoUnits(units, "/run/comquad/secrets")

	if len(result) != 3 {
		t.Fatalf("expected 3 units, got %d", len(result))
	}

	if result[0].Type != c2qtypes.UnitContainer {
		t.Error("expected first unit to be container")
	}
	if result[1].Type != c2qtypes.UnitCQSecret {
		t.Error("expected second unit to be cq-secret (preserved type)")
	}
	if result[1].Name != "db_password" {
		t.Error("expected second unit name to be db_password")
	}
	if result[2].Type != c2qtypes.UnitNetwork {
		t.Error("expected third unit to be network")
	}
}
