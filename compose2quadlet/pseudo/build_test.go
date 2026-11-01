package pseudo

import (
	"strings"
	"testing"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestConvertBuild_Basic(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQBuild,
		Name: "app",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionBuild,
				Directives: []c2qtypes.Directive{
					{Key: "SetWorkingDirectory", Values: []string{"/app"}},
					{Key: "ImageTag", Values: []string{"myapp_app:latest"}},
				},
			},
		},
	}

	result := convertBuild(unit)

	if result.Type != c2qtypes.UnitCQBuild {
		t.Fatalf("expected cq-build type, got %s", result.Type)
	}
	if result.Name != "app" {
		t.Fatalf("expected name app, got %s", result.Name)
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

	hasType := false
	hasRemain := false
	hasWorkDir := false
	hasExec := false
	for _, d := range serviceSection.Directives {
		if d.Key == "Type" && len(d.Values) > 0 && d.Values[0] == "oneshot" {
			hasType = true
		}
		if d.Key == "RemainAfterExit" && len(d.Values) > 0 && d.Values[0] == "yes" {
			hasRemain = true
		}
		if d.Key == "WorkingDirectory" && len(d.Values) > 0 && d.Values[0] == "/app" {
			hasWorkDir = true
		}
		if d.Key == "ExecStart" && len(d.Values) > 0 {
			if strings.Contains(d.Values[0], "podman build") && strings.Contains(d.Values[0], "-t myapp_app:latest") {
				hasExec = true
			}
		}
	}
	if !hasType {
		t.Error("expected Type=oneshot")
	}
	if !hasRemain {
		t.Error("expected RemainAfterExit=yes")
	}
	if !hasWorkDir {
		t.Error("expected WorkingDirectory=/app")
	}
	if !hasExec {
		t.Error("expected ExecStart with podman build -t myapp_app:latest")
	}
}

func TestConvertBuild_WithDockerfile(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQBuild,
		Name: "app",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionBuild,
				Directives: []c2qtypes.Directive{
					{Key: "SetWorkingDirectory", Values: []string{"/app"}},
					{Key: "File", Values: []string{"Dockerfile.prod"}},
					{Key: "ImageTag", Values: []string{"myapp:latest"}},
				},
			},
		},
	}

	result := convertBuild(unit)

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	for _, d := range serviceSection.Directives {
		if d.Key == "ExecStart" && len(d.Values) > 0 {
			if !strings.Contains(d.Values[0], "-f Dockerfile.prod") {
				t.Error("expected ExecStart with -f Dockerfile.prod")
			}
		}
	}
}

func TestConvertBuild_WithNoCache(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQBuild,
		Name: "app",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionBuild,
				Directives: []c2qtypes.Directive{
					{Key: "SetWorkingDirectory", Values: []string{"/app"}},
					{Key: "ImageTag", Values: []string{"myapp:latest"}},
					{Key: "PodmanArgs", Values: []string{"--no-cache"}},
				},
			},
		},
	}

	result := convertBuild(unit)

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	for _, d := range serviceSection.Directives {
		if d.Key == "ExecStart" && len(d.Values) > 0 {
			if !strings.Contains(d.Values[0], "--no-cache") {
				t.Error("expected ExecStart with --no-cache")
			}
		}
	}
}

func TestConvertBuild_NoWorkDir(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQBuild,
		Name: "app",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionBuild,
				Directives: []c2qtypes.Directive{
					{Key: "ImageTag", Values: []string{"myapp:latest"}},
				},
			},
		},
	}

	result := convertBuild(unit)

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	for _, d := range serviceSection.Directives {
		if d.Key == "WorkingDirectory" {
			t.Error("should not have WorkingDirectory when not specified")
		}
	}
}
