package pseudo

import (
	"testing"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestConvertImage_Basic(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQImage,
		Name: "web",
		Sections: []c2qtypes.Section{
			{
				Name: c2qtypes.SectionImage,
				Directives: []c2qtypes.Directive{
					{Key: "Image", Values: []string{"docker.io/library/nginx:latest"}},
				},
			},
		},
	}

	result := convertImage(unit)

	if result.Type != c2qtypes.UnitCQImage {
		t.Fatalf("expected cq-image type, got %s", result.Type)
	}
	if result.Name != "web" {
		t.Fatalf("expected name web, got %s", result.Name)
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
	hasExec := false
	for _, d := range serviceSection.Directives {
		if d.Key == "Type" && len(d.Values) > 0 && d.Values[0] == "oneshot" {
			hasType = true
		}
		if d.Key == "RemainAfterExit" && len(d.Values) > 0 && d.Values[0] == "yes" {
			hasRemain = true
		}
		if d.Key == "ExecStart" && len(d.Values) > 0 && d.Values[0] == "/usr/bin/podman pull docker.io/library/nginx:latest" {
			hasExec = true
		}
	}
	if !hasType {
		t.Error("expected Type=oneshot")
	}
	if !hasRemain {
		t.Error("expected RemainAfterExit=yes")
	}
	if !hasExec {
		t.Error("expected ExecStart=/usr/bin/podman pull docker.io/library/nginx:latest")
	}
}

func TestConvertImage_NoImage(t *testing.T) {
	unit := c2qtypes.QuadletUnit{
		Type: c2qtypes.UnitCQImage,
		Name: "web",
		Sections: []c2qtypes.Section{
			{
				Name:       c2qtypes.SectionImage,
				Directives: []c2qtypes.Directive{},
			},
		},
	}

	result := convertImage(unit)

	var serviceSection *c2qtypes.Section
	for i := range result.Sections {
		if result.Sections[i].Name == c2qtypes.SectionService {
			serviceSection = &result.Sections[i]
			break
		}
	}

	for _, d := range serviceSection.Directives {
		if d.Key == "ExecStart" {
			t.Error("should not have ExecStart when no image specified")
		}
	}
}
