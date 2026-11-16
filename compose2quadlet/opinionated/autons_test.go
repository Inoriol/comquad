package opinionated

import (
	"strings"
	"testing"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestApplyAutoNS_Disabled(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		{
			Type: c2qtypes.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2qtypes.Section{
				{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
					{Key: "Image", Values: []string{"nginx"}},
				}},
			},
		},
	}
	cfg := &c2qtypes.Config{}

	result := ApplyAutoNS(units, cfg)

	if len(result) != 1 {
		t.Fatal("expected 1 unit")
	}
	for _, d := range result[0].Sections[0].Directives {
		if d.Key == "PodmanArgs" {
			t.Error("expected no PodmanArgs when AutoNS is disabled")
		}
	}
}

func TestApplyAutoNS_AnchorContainer(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		{
			Type: c2qtypes.UnitContainer,
			Name: "cq-myapp-db",
			Sections: []c2qtypes.Section{
				{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
					{Key: "Image", Values: []string{"postgres"}},
					{Key: "ContainerName", Values: []string{"myapp-db"}},
				}},
			},
		},
	}
	cfg := &c2qtypes.Config{
		FilePrefix:  "cq-",
		ProjectName: "myapp",
		AutoNS: &c2qtypes.AutoNSConfig{
			StackBaseUID:  65535,
			AnchorService: "db",
		},
	}

	result := ApplyAutoNS(units, cfg)

	containerSec := findSection(result[0], c2qtypes.SectionContainer)
	if containerSec == nil {
		t.Fatal("expected [Container] section")
	}

	found := false
	for _, d := range containerSec.Directives {
		if d.Key == "PodmanArgs" && len(d.Values) > 0 {
			if strings.Contains(d.Values[0], "--userns=auto:uidmapping=0:65535:65535") {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected anchor to have --userns=auto:uidmapping=0:65535:65535")
	}
}

func TestApplyAutoNS_NonAnchorJoinsNamespace(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		{
			Type: c2qtypes.UnitContainer,
			Name: "cq-myapp-db",
			Sections: []c2qtypes.Section{
				{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
					{Key: "ContainerName", Values: []string{"myapp-db"}},
				}},
			},
		},
		{
			Type: c2qtypes.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2qtypes.Section{
				{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
					{Key: "ContainerName", Values: []string{"myapp-web"}},
				}},
			},
		},
	}
	cfg := &c2qtypes.Config{
		FilePrefix:  "cq-",
		ProjectName: "myapp",
		AutoNS: &c2qtypes.AutoNSConfig{
			StackBaseUID:  65535,
			AnchorService: "db",
		},
	}

	result := ApplyAutoNS(units, cfg)

	webUnit := findUnit(result, "cq-myapp-web")
	if webUnit == nil {
		t.Fatal("expected web unit")
	}

	containerSec := findSection(*webUnit, c2qtypes.SectionContainer)
	if containerSec == nil {
		t.Fatal("expected [Container] section in web unit")
	}

	foundJoin := false
	for _, d := range containerSec.Directives {
		if d.Key == "PodmanArgs" && len(d.Values) > 0 {
			if d.Values[0] == "--userns=container:myapp-db" {
				foundJoin = true
			}
		}
	}
	if !foundJoin {
		t.Error("expected non-anchor to have --userns=container:myapp-db")
	}

	unitSec := findSection(*webUnit, c2qtypes.SectionUnit)
	if unitSec == nil {
		t.Fatal("expected [Unit] section in web unit")
	}

	hasAfter := false
	hasRequires := false
	for _, d := range unitSec.Directives {
		if d.Key == "After" && len(d.Values) > 0 && d.Values[0] == "cq-myapp-db.service" {
			hasAfter = true
		}
		if d.Key == "Requires" && len(d.Values) > 0 && d.Values[0] == "cq-myapp-db.service" {
			hasRequires = true
		}
	}
	if !hasAfter {
		t.Error("expected non-anchor to have After=cq-myapp-db.service")
	}
	if !hasRequires {
		t.Error("expected non-anchor to have Requires=cq-myapp-db.service")
	}
}

func TestApplyAutoNS_VolumeIdmap(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		{
			Type: c2qtypes.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2qtypes.Section{
				{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
					{Key: "ContainerName", Values: []string{"myapp-web"}},
					{Key: "Volume", Values: []string{"/host/data:/container/data:z"}},
				}},
			},
		},
	}
	cfg := &c2qtypes.Config{
		FilePrefix:  "cq-",
		ProjectName: "myapp",
		AutoNS: &c2qtypes.AutoNSConfig{
			StackBaseUID:  65535,
			AnchorService: "web",
		},
	}

	result := ApplyAutoNS(units, cfg)

	containerSec := findSection(result[0], c2qtypes.SectionContainer)
	for _, d := range containerSec.Directives {
		if d.Key == "Volume" && len(d.Values) > 0 {
			if !strings.Contains(d.Values[0], "idmap") {
				t.Errorf("expected volume to have idmap option, got %q", d.Values[0])
			}
		}
	}
}

func TestApplyAutoNS_MountIdmap(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		{
			Type: c2qtypes.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2qtypes.Section{
				{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
					{Key: "ContainerName", Values: []string{"myapp-web"}},
					{Key: "Mount", Values: []string{"type=bind,source=/host/data,target=/container/data"}},
				}},
			},
		},
	}
	cfg := &c2qtypes.Config{
		FilePrefix:  "cq-",
		ProjectName: "myapp",
		AutoNS: &c2qtypes.AutoNSConfig{
			StackBaseUID:  65535,
			AnchorService: "web",
		},
	}

	result := ApplyAutoNS(units, cfg)

	containerSec := findSection(result[0], c2qtypes.SectionContainer)
	for _, d := range containerSec.Directives {
		if d.Key == "Mount" && len(d.Values) > 0 {
			if !strings.Contains(d.Values[0], "idmap") {
				t.Errorf("expected mount to have idmap option, got %q", d.Values[0])
			}
		}
	}
}

func TestApplyAutoNS_NoDoubleIdmap(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		{
			Type: c2qtypes.UnitContainer,
			Name: "cq-myapp-web",
			Sections: []c2qtypes.Section{
				{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
					{Key: "ContainerName", Values: []string{"myapp-web"}},
					{Key: "Volume", Values: []string{"/host/data:/container/data:idmap"}},
				}},
			},
		},
	}
	cfg := &c2qtypes.Config{
		FilePrefix:  "cq-",
		ProjectName: "myapp",
		AutoNS: &c2qtypes.AutoNSConfig{
			StackBaseUID:  65535,
			AnchorService: "web",
		},
	}

	result := ApplyAutoNS(units, cfg)

	containerSec := findSection(result[0], c2qtypes.SectionContainer)
	for _, d := range containerSec.Directives {
		if d.Key == "Volume" && len(d.Values) > 0 {
			count := strings.Count(d.Values[0], "idmap")
			if count != 1 {
				t.Errorf("expected exactly one idmap occurrence, got %d in %q", count, d.Values[0])
			}
		}
	}
}

func findUnit(units []c2qtypes.QuadletUnit, name string) *c2qtypes.QuadletUnit {
	for i := range units {
		if units[i].Name == name {
			return &units[i]
		}
	}
	return nil
}

func findSection(unit c2qtypes.QuadletUnit, sectionName string) *c2qtypes.Section {
	for i := range unit.Sections {
		if unit.Sections[i].Name == sectionName {
			return &unit.Sections[i]
		}
	}
	return nil
}
