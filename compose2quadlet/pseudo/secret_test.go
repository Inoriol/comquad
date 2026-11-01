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

	result := convertSecret(unit)

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

	hasLoadCredential := false
	for _, d := range serviceSection.Directives {
		if d.Key == "LoadCredential" && len(d.Values) > 0 {
			hasLoadCredential = true
			expected := "db_password:/bin/sh -c 'printf \"%s\" \"$DB_PASSWORD\"'"
			if d.Values[0] != expected {
				t.Errorf("expected LoadCredential=%s, got %s", expected, d.Values[0])
			}
		}
	}

	if !hasLoadCredential {
		t.Error("expected LoadCredential directive")
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

	result := convertSecret(unit)

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

	hasLoadCredential := false
	for _, d := range serviceSection.Directives {
		if d.Key == "LoadCredential" && len(d.Values) > 0 {
			hasLoadCredential = true
			expected := "api_key:/etc/secrets/api.key"
			if d.Values[0] != expected {
				t.Errorf("expected LoadCredential=%s, got %s", expected, d.Values[0])
			}
		}
	}

	if !hasLoadCredential {
		t.Error("expected LoadCredential directive")
	}
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

	result := ConvertPseudoUnits(units)

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
