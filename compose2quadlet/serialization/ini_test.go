package serialization

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	c2q "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestMarshal_EmptyUnit(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitContainer,
		Name: "empty",
	}
	out := Marshal(unit)
	if out != "" {
		t.Fatalf("expected empty string, got %q", out)
	}
}

func TestMarshal_SingleSection(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitContainer,
		Name: "web",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionContainer,
				Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"nginx"}},
					{Key: "PublishPort", Values: []string{"8080:80"}},
				},
			},
		},
	}
	out := Marshal(unit)
	expected := "[Container]\nImage=nginx\nPublishPort=8080:80\n"
	if out != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, out)
	}
}

func TestMarshal_MultiValueDirective(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitContainer,
		Name: "web",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionContainer,
				Directives: []c2q.Directive{
					{Key: "Volume", Values: []string{"/data:/data:Z", "/tmp:/tmp"}},
				},
			},
		},
	}
	out := Marshal(unit)
	expected := "[Container]\nVolume=/data:/data:Z\nVolume=/tmp:/tmp\n"
	if out != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, out)
	}
}

func TestMarshal_BooleanFlag(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitContainer,
		Name: "web",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionContainer,
				Directives: []c2q.Directive{
					{Key: "NoNewPrivileges"},
				},
			},
		},
	}
	out := Marshal(unit)
	expected := "[Container]\nNoNewPrivileges=\n"
	if out != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, out)
	}
}

func TestMarshal_MultipleSections(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitContainer,
		Name: "web",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionUnit,
				Directives: []c2q.Directive{
					{Key: "Requires", Values: []string{"db.service"}},
				},
			},
			{
				Name: c2q.SectionContainer,
				Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"nginx"}},
				},
			},
			{
				Name: c2q.SectionInstall,
				Directives: []c2q.Directive{
					{Key: "WantedBy", Values: []string{"default.target"}},
				},
			},
		},
	}
	out := Marshal(unit)
	expected := "[Unit]\nRequires=db.service\n\n[Container]\nImage=nginx\n\n[Install]\nWantedBy=default.target\n"
	if out != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, out)
	}
}

func TestMarshal_EmptySectionsOmitted(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitContainer,
		Name: "web",
		Sections: []c2q.Section{
			{Name: c2q.SectionUnit},
			{
				Name: c2q.SectionContainer,
				Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"nginx"}},
				},
			},
			{Name: c2q.SectionInstall},
		},
	}
	out := Marshal(unit)
	expected := "[Container]\nImage=nginx\n"
	if out != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, out)
	}
}

func TestWrite_MultipleUnits(t *testing.T) {
	units := []c2q.QuadletUnit{
		{
			Type: c2q.UnitContainer,
			Name: "web",
			Sections: []c2q.Section{
				{
					Name: c2q.SectionContainer,
					Directives: []c2q.Directive{
						{Key: "Image", Values: []string{"nginx"}},
					},
				},
			},
		},
		{
			Type: c2q.UnitContainer,
			Name: "db",
			Sections: []c2q.Section{
				{
					Name: c2q.SectionContainer,
					Directives: []c2q.Directive{
						{Key: "Image", Values: []string{"postgres"}},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := Write(&buf, units)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	expected := "[Container]\nImage=nginx\n\n[Container]\nImage=postgres\n"
	if out != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, out)
	}
}

func TestRoundTrip_Container(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitContainer,
		Name: "web",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionUnit,
				Directives: []c2q.Directive{
					{Key: "Requires", Values: []string{"db.service"}},
				},
			},
			{
				Name: c2q.SectionContainer,
				Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"nginx"}},
					{Key: "PublishPort", Values: []string{"8080:80", "8443:443"}},
					{Key: "NoNewPrivileges"},
				},
			},
			{
				Name: c2q.SectionInstall,
				Directives: []c2q.Directive{
					{Key: "WantedBy", Values: []string{"default.target"}},
				},
			},
		},
	}

	out := Marshal(unit)
	parsed := Unmarshal(out, c2q.UnitContainer, "web")

	if len(parsed.Sections) != len(unit.Sections) {
		t.Fatalf("section count mismatch: %d != %d", len(parsed.Sections), len(unit.Sections))
	}
	for i, sec := range unit.Sections {
		ps := parsed.Sections[i]
		if ps.Name != sec.Name {
			t.Fatalf("section name mismatch: %s != %s", ps.Name, sec.Name)
		}
		if len(ps.Directives) != len(sec.Directives) {
			t.Fatalf("directive count mismatch in [%s]: %d != %d", sec.Name, len(ps.Directives), len(sec.Directives))
		}
		for j, d := range sec.Directives {
			pd := ps.Directives[j]
			if pd.Key != d.Key {
				t.Fatalf("directive key mismatch: %s != %s", pd.Key, d.Key)
			}
			if len(pd.Values) != len(d.Values) {
				t.Fatalf("value count mismatch for %s: %v != %v", d.Key, pd.Values, d.Values)
			}
			for k, v := range d.Values {
				if pd.Values[k] != v {
					t.Fatalf("value mismatch for %s[%d]: %s != %s", d.Key, k, pd.Values[k], v)
				}
			}
		}
	}
}

func TestMarshal_NetworkUnit(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitNetwork,
		Name: "backend",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionNetwork,
				Directives: []c2q.Directive{
					{Key: "Subnet", Values: []string{"10.0.0.0/24"}},
					{Key: "Gateway", Values: []string{"10.0.0.1"}},
				},
			},
		},
	}
	out := Marshal(unit)
	expected := "[Network]\nSubnet=10.0.0.0/24\nGateway=10.0.0.1\n"
	if out != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, out)
	}
}

func TestMarshal_VolumeUnit(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitVolume,
		Name: "data",
		Sections: []c2q.Section{
			{
				Name:       c2q.SectionVolume,
				Directives: []c2q.Directive{},
			},
		},
	}
	out := Marshal(unit)
	if out != "" {
		t.Fatalf("expected empty output for empty section, got %q", out)
	}
}

func TestMarshal_ImageUnit(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitImage,
		Name: "nginx",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionImage,
				Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx:latest"}},
				},
			},
		},
	}
	out := Marshal(unit)
	expected := "[Image]\nImage=docker.io/library/nginx:latest\n"
	if out != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, out)
	}
}

func TestMarshal_IgnoresCommentLines(t *testing.T) {
	data := `# FileName=web.container
[Container]
Image=nginx
`
	parsed := Unmarshal(data, c2q.UnitContainer, "web")
	if len(parsed.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(parsed.Sections))
	}
	d := parsed.Sections[0].Directives[0]
	if d.Key != "Image" || d.Values[0] != "nginx" {
		t.Fatalf("unexpected directive: %s=%v", d.Key, d.Values)
	}
}

func TestGoldenFiles(t *testing.T) {
	tests := []struct {
		name string
		unit c2q.QuadletUnit
	}{
		{
			"container",
			c2q.QuadletUnit{
				Type: c2q.UnitContainer,
				Name: "web",
				Sections: []c2q.Section{
					{
						Name: c2q.SectionUnit,
						Directives: []c2q.Directive{
							{Key: "Requires", Values: []string{"db.service"}},
							{Key: "After", Values: []string{"db.service"}},
						},
					},
					{
						Name: c2q.SectionContainer,
						Directives: []c2q.Directive{
							{Key: "Image", Values: []string{"nginx.image"}},
							{Key: "PublishPort", Values: []string{"8080:80", "8443:443"}},
							{Key: "NoNewPrivileges"},
							{Key: "Environment", Values: []string{"NODE_ENV=production"}},
						},
					},
					{
						Name: c2q.SectionService,
						Directives: []c2q.Directive{
							{Key: "Restart", Values: []string{"always"}},
						},
					},
					{
						Name: c2q.SectionInstall,
						Directives: []c2q.Directive{
							{Key: "WantedBy", Values: []string{"default.target"}},
						},
					},
				},
			},
		},
		{
			"network",
			c2q.QuadletUnit{
				Type: c2q.UnitNetwork,
				Name: "backend",
				Sections: []c2q.Section{
					{
						Name: c2q.SectionNetwork,
						Directives: []c2q.Directive{
							{Key: "Subnet", Values: []string{"10.0.0.0/24"}},
							{Key: "Gateway", Values: []string{"10.0.0.1"}},
							{Key: "IPv6", Values: []string{"true"}},
						},
					},
				},
			},
		},
		{
			"volume",
			c2q.QuadletUnit{
				Type: c2q.UnitVolume,
				Name: "data",
				Sections: []c2q.Section{
					{
						Name: c2q.SectionVolume,
						Directives: []c2q.Directive{
							{Key: "Driver", Values: []string{"local"}},
						},
					},
				},
			},
		},
		{
			"image",
			c2q.QuadletUnit{
				Type: c2q.UnitImage,
				Name: "nginx",
				Sections: []c2q.Section{
					{
						Name: c2q.SectionImage,
						Directives: []c2q.Directive{
							{Key: "Image", Values: []string{"docker.io/library/nginx:latest"}},
						},
					},
				},
			},
		},
		{
			"build",
			c2q.QuadletUnit{
				Type: c2q.UnitBuild,
				Name: "myapp",
				Sections: []c2q.Section{
					{
						Name: c2q.SectionBuild,
						Directives: []c2q.Directive{
							{Key: "SetWorkingDirectory", Values: []string{"."}},
							{Key: "File", Values: []string{"Dockerfile"}},
							{Key: "Target", Values: []string{"production"}},
							{Key: "ImageTag", Values: []string{"myapp:latest"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := Marshal(tt.unit)
			goldenPath := filepath.Join("..", "testdata", "serialization", tt.name+".golden")
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}
			if out != string(want) {
				t.Fatalf("marshal output differs from golden file:\nexpected:\n%q\ngot:\n%q", string(want), out)
			}
		})
	}
}

func TestRoundTrip_Network(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitNetwork,
		Name: "backend",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionNetwork,
				Directives: []c2q.Directive{
					{Key: "Subnet", Values: []string{"10.0.0.0/24"}},
					{Key: "Gateway", Values: []string{"10.0.0.1"}},
					{Key: "IPv6", Values: []string{"true"}},
				},
			},
		},
	}
	out := Marshal(unit)
	parsed := Unmarshal(out, c2q.UnitNetwork, "backend")
	roundTripCheck(t, unit, parsed)
}

func TestRoundTrip_Volume(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitVolume,
		Name: "data",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionVolume,
				Directives: []c2q.Directive{
					{Key: "Driver", Values: []string{"nfs"}},
					{Key: "Options", Values: []string{"type=nfs", "o=addr=10.0.0.1"}},
				},
			},
		},
	}
	out := Marshal(unit)
	parsed := Unmarshal(out, c2q.UnitVolume, "data")
	roundTripCheck(t, unit, parsed)
}

func TestRoundTrip_Image(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitImage,
		Name: "nginx",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionImage,
				Directives: []c2q.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx:latest"}},
					{Key: "AutoUpdate", Values: []string{"registry"}},
				},
			},
		},
	}
	out := Marshal(unit)
	parsed := Unmarshal(out, c2q.UnitImage, "nginx")
	roundTripCheck(t, unit, parsed)
}

func TestRoundTrip_Build(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitBuild,
		Name: "myapp",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionBuild,
				Directives: []c2q.Directive{
					{Key: "SetWorkingDirectory", Values: []string{"."}},
					{Key: "File", Values: []string{"Dockerfile"}},
					{Key: "Target", Values: []string{"production"}},
					{Key: "ImageTag", Values: []string{"myapp:latest", "myapp:v2"}},
					{Key: "PodmanArgs", Values: []string{"--no-cache"}},
				},
			},
		},
	}
	out := Marshal(unit)
	parsed := Unmarshal(out, c2q.UnitBuild, "myapp")
	roundTripCheck(t, unit, parsed)
}

func TestRoundTrip_EmptyDirective(t *testing.T) {
	unit := c2q.QuadletUnit{
		Type: c2q.UnitContainer,
		Name: "web",
		Sections: []c2q.Section{
			{
				Name: c2q.SectionContainer,
				Directives: []c2q.Directive{
					{Key: "NoNewPrivileges"},
					{Key: "SecurityLabelDisable"},
				},
			},
		},
	}
	out := Marshal(unit)
	parsed := Unmarshal(out, c2q.UnitContainer, "web")
	roundTripCheck(t, unit, parsed)
}

func roundTripCheck(t *testing.T, want, got c2q.QuadletUnit) {
	t.Helper()
	if len(got.Sections) != len(want.Sections) {
		t.Fatalf("section count mismatch: %d != %d", len(got.Sections), len(want.Sections))
	}
	for i, sec := range want.Sections {
		ps := got.Sections[i]
		if ps.Name != sec.Name {
			t.Fatalf("section name mismatch: %s != %s", ps.Name, sec.Name)
		}
		if len(ps.Directives) != len(sec.Directives) {
			t.Fatalf("directive count mismatch in [%s]: %d != %d", sec.Name, len(ps.Directives), len(sec.Directives))
		}
		for j, d := range sec.Directives {
			pd := ps.Directives[j]
			if pd.Key != d.Key {
				t.Fatalf("directive key mismatch: %s != %s", pd.Key, d.Key)
			}
			if len(pd.Values) != len(d.Values) {
				t.Fatalf("value count mismatch for %s: %v != %v", d.Key, pd.Values, d.Values)
			}
			for k, v := range d.Values {
				if pd.Values[k] != v {
					t.Fatalf("value mismatch for %s[%d]: %s != %s", d.Key, k, pd.Values[k], v)
				}
			}
		}
	}
}
