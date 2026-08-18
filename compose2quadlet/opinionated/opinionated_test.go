package opinionated

import (
	"testing"

	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func mkUnit(typ c2qtypes.UnitType, name string, sections []c2qtypes.Section) c2qtypes.QuadletUnit {
	return c2qtypes.QuadletUnit{Type: typ, Name: name, Sections: sections}
}

func mkDir(key string, values ...string) c2qtypes.Directive {
	return c2qtypes.Directive{Key: key, Values: values}
}

func hasSection(unit c2qtypes.QuadletUnit, name string) bool {
	for _, s := range unit.Sections {
		if s.Name == name {
			return true
		}
	}
	return false
}

func assertDirectiveValue(t *testing.T, sec c2qtypes.Section, key, want string) {
	t.Helper()
	for _, d := range sec.Directives {
		if d.Key == key {
			for _, v := range d.Values {
				if v == want {
					return
				}
			}
		}
	}
	t.Fatalf("directive %s=%s not found in section %s", key, want, sec.Name)
}

func TestApplyPrefix(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", nil),
		mkUnit(c2qtypes.UnitNetwork, "backend", nil),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.ProjectName = "myapp"

	result := ApplyPrefix(units, cfg)
	if result[0].Name != "cq-myapp-web" {
		t.Fatalf("expected cq-myapp-web, got %s", result[0].Name)
	}
	if result[1].Name != "cq-myapp-backend" {
		t.Fatalf("expected cq-myapp-backend, got %s", result[1].Name)
	}
}

func TestApplyPrefix_NoProjectName(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", nil),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.ProjectName = ""

	result := ApplyPrefix(units, cfg)
	if result[0].Name != "cq-web" {
		t.Fatalf("expected cq-web, got %s", result[0].Name)
	}
}

func TestApplyPrefix_NoOp(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", nil),
	}
	cfg := &c2qtypes.Config{}

	result := ApplyPrefix(units, cfg)
	if result[0].Name != "web" {
		t.Fatalf("expected web, got %s", result[0].Name)
	}
}

func TestApplyReferences_NetworkVolume(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "prefix-web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Network", "backend.network"),
				mkDir("Volume", "data.volume"),
				mkDir("Image", "nginx.image"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.FilePrefix = "cq-"

	result := ApplyReferences(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "Network", "cq-backend.network")
	assertDirectiveValue(t, sec, "Volume", "cq-data.volume")
	assertDirectiveValue(t, sec, "Image", "cq-nginx.image")
}

func TestApplyReferences_ExternalNetworkAndVolume(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "cq-app-web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Network", "backend.network"),
				mkDir("Volume", "data.volume:/var/lib/data:rw"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.ProjectName = "app"
	cfg.ExternalNetworks = map[string]string{"backend": "shared-net"}
	cfg.ExternalVolumes = map[string]string{"data": "shared-data"}

	result := ApplyReferences(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "Network", "shared-net")
	assertDirectiveValue(t, sec, "Volume", "shared-data:/var/lib/data:rw")
}

func TestApplyReferences_UnitDeps(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "prefixed-web", []c2qtypes.Section{
			{Name: c2qtypes.SectionUnit, Directives: []c2qtypes.Directive{
				mkDir("After", "db"),
				mkDir("Requires", "redis"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplyReferences(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "After", "cq-db.container")
	assertDirectiveValue(t, sec, "Requires", "cq-redis.container")
}

func TestApplyNetworkAliases(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Network", "backend.network"),
				mkDir("Network", "frontend.network"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplyNetworkAliases(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "NetworkAlias", "web")
}

func TestApplyNetworkAliases_Disabled(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Network", "backend.network"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.NetworkAliases = false

	result := ApplyNetworkAliases(units, cfg)
	sec := result[0].Sections[0]
	for _, d := range sec.Directives {
		if d.Key == "NetworkAlias" {
			t.Fatal("expected no NetworkAlias when disabled")
		}
	}
}

func TestApplySELinux_Volume(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Volume", "/data:/data"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplySELinux(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "Volume", "/data:/data,z")
}

func TestApplySELinux_Mount(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Mount", "type=bind,source=/data,destination=/data"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplySELinux(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "Mount", "type=bind,source=/data,destination=/data,relabel=shared")
}

func TestApplySELinux_AlreadySet(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Volume", "/data:/data:Z"),
				mkDir("Volume", "/logs:/logs:z"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplySELinux(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "Volume", "/data:/data:Z")
	assertDirectiveValue(t, sec, "Volume", "/logs:/logs:z")
}

func TestApplySELinux_NoFalsePositive(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Volume", "/data:zoo:/mnt"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplySELinux(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "Volume", "/data:zoo:/mnt,z")
}

func TestApplyLabels(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Image", "nginx.image"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.Labels = map[string]string{"com.example.managed": "true"}

	result := ApplyLabels(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "Label", "com.example.managed=true")
}

func TestApplyLabels_NoDuplication(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Label", "com.example.managed=true"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.Labels = map[string]string{"com.example.managed": "true", "com.example.new": "yes"}

	result := ApplyLabels(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "Label", "com.example.managed=true")
	assertDirectiveValue(t, sec, "Label", "com.example.new=yes")

	count := 0
	for _, d := range sec.Directives {
		if d.Key == "Label" {
			count += len(d.Values)
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 Label values, got %d", count)
	}
}

func TestApplyLabels_SortedOrder(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Image", "nginx.image"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.Labels = map[string]string{
		"com.comquad.project": "myapp",
		"com.comquad.managed": "true",
	}

	result := ApplyLabels(units, cfg)
	sec := result[0].Sections[0]

	var labels []string
	for _, d := range sec.Directives {
		if d.Key == "Label" {
			labels = append(labels, d.Values...)
		}
	}

	want := []string{"com.comquad.managed=true", "com.comquad.project=myapp"}
	if len(labels) != len(want) {
		t.Fatalf("expected labels %v, got %v", want, labels)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("expected sorted labels %v, got %v", want, labels)
		}
	}
}

func TestApplyDefaultNetwork(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Image", "nginx.image"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.ProjectName = "app"

	result := ApplyDefaultNetwork(units, cfg)
	if len(result) != 2 {
		t.Fatalf("expected 2 units, got %d", len(result))
	}
	netUnit := result[1]
	if netUnit.Type != c2qtypes.UnitNetwork || netUnit.Name != "cq-app-default" {
		t.Fatalf("expected network unit cq-app-default, got %s %s", netUnit.Name, netUnit.Type)
	}
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "Network", "cq-app-default.network")
}

func TestApplyDefaultNetwork_AlreadyHasNetwork(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Network", "backend.network"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplyDefaultNetwork(units, cfg)
	if len(result) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(result))
	}
}

func TestApplyDefaultNetwork_MixedExplicitAndImplicitNetworks(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Network", "backend.network"),
			}},
		}),
		mkUnit(c2qtypes.UnitContainer, "worker", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Image", "worker.image"),
			}},
		}),
		mkUnit(c2qtypes.UnitNetwork, "backend", nil),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.ProjectName = "app"

	result := ApplyDefaultNetwork(units, cfg)
	if len(result) != 4 {
		t.Fatalf("expected existing and default network plus two containers, got %d units", len(result))
	}
	sec := result[1].Sections[0]
	assertDirectiveValue(t, sec, "Network", "cq-app-default.network")
}

func TestApplyPortOffset(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("PublishPort", "80:80"),
				mkDir("PublishPort", "192.168.1.1:80:80"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PortOffset = 10000

	result := ApplyPortOffset(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "PublishPort", "10080:80")
	assertDirectiveValue(t, sec, "PublishPort", "192.168.1.1:10080:80")
}

func TestApplyPortOffset_NonPrivileged_Unchanged(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("PublishPort", "8080:80"),
				mkDir("PublishPort", "192.168.1.1:8080:80"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PortOffset = 10000

	result := ApplyPortOffset(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "PublishPort", "8080:80")
	assertDirectiveValue(t, sec, "PublishPort", "192.168.1.1:8080:80")
}

func TestApplyPortOffset_WithProtocol(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("PublishPort", "80:80/udp"),
				mkDir("PublishPort", "80/tcp"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PortOffset = 10000

	result := ApplyPortOffset(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "PublishPort", "10080:80/udp")
	assertDirectiveValue(t, sec, "PublishPort", "80/tcp")
}

func TestApplyPortOffset_Zero(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("PublishPort", "8080:80"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplyPortOffset(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "PublishPort", "8080:80")
}

func TestApplyAutoUpdate(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Image", "nginx.image"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.AutoUpdate = true

	result := ApplyAutoUpdate(units, cfg)
	sec := result[0].Sections[0]
	assertDirectiveValue(t, sec, "AutoUpdate", "registry")
}

func TestApplyAutoUpdate_Disabled(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplyAutoUpdate(units, cfg)
	sec := result[0].Sections[0]
	for _, d := range sec.Directives {
		if d.Key == "AutoUpdate" {
			t.Fatal("expected no AutoUpdate when disabled")
		}
	}
}

func TestApplyInstallSection(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Image", "nginx.image"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplyInstallSection(units, cfg)
	if !hasSection(result[0], c2qtypes.SectionInstall) {
		t.Fatal("expected [Install] section")
	}
	assertDirectiveValue(t, result[0].Sections[1], "WantedBy", "default.target")
}

func TestApplyInstallSection_AlreadyHas(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Image", "nginx.image"),
			}},
			{Name: c2qtypes.SectionInstall, Directives: []c2qtypes.Directive{
				mkDir("WantedBy", "multi-user.target"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()

	result := ApplyInstallSection(units, cfg)
	if result[0].Sections[1].Directives[0].Values[0] != "multi-user.target" {
		t.Fatal("existing [Install] section should not be modified")
	}
}

func TestApplyInstallSection_Disabled(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.InstallSection = false

	result := ApplyInstallSection(units, cfg)
	if hasSection(result[0], c2qtypes.SectionInstall) {
		t.Fatal("expected no [Install] section when disabled")
	}
}

func TestApply_FullPipeline(t *testing.T) {
	units := []c2qtypes.QuadletUnit{
		mkUnit(c2qtypes.UnitContainer, "web", []c2qtypes.Section{
			{Name: c2qtypes.SectionContainer, Directives: []c2qtypes.Directive{
				mkDir("Image", "nginx.image"),
				mkDir("Network", "backend.network"),
				mkDir("PublishPort", "8080:80"),
			}},
		}),
		mkUnit(c2qtypes.UnitNetwork, "backend", []c2qtypes.Section{
			{Name: c2qtypes.SectionNetwork, Directives: []c2qtypes.Directive{
				mkDir("Subnet", "10.0.0.0/24"),
			}},
		}),
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.ProjectName = "myapp"
	cfg.PortOffset = 10000
	cfg.AutoUpdate = true
	cfg.Labels = map[string]string{"managed": "true"}

	result := Apply(units, cfg)

	if result[0].Name != "cq-myapp-web" {
		t.Fatalf("expected cq-myapp-web, got %s", result[0].Name)
	}
	if result[1].Name != "cq-myapp-backend" {
		t.Fatalf("expected cq-myapp-backend, got %s", result[1].Name)
	}

	containerSec := result[0].Sections[0]
	assertDirectiveValue(t, containerSec, "Network", "cq-myapp-backend.network")
	assertDirectiveValue(t, containerSec, "NetworkAlias", "web")
	assertDirectiveValue(t, containerSec, "NetworkAlias", "myapp-web")
	assertDirectiveValue(t, containerSec, "PublishPort", "8080:80")
	assertDirectiveValue(t, containerSec, "AutoUpdate", "registry")
	assertDirectiveValue(t, containerSec, "Label", "managed=true")

	if !hasSection(result[0], c2qtypes.SectionInstall) {
		t.Fatal("expected [Install] section")
	}
}

func TestHasSELinuxContext(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"/data:/data:z", true},
		{"/data:/data:Z", true},
		{"/data:/data:z,ro", true},
		{"/data:/data:Z,uid=1000", true},
		{"/data:/data", false},
		{"/data:zoo:/mnt", false},
		{"/data:/data:zz", false},
		{"", false},
	}
	for _, tc := range tests {
		if hasSELinuxContext(tc.in) != tc.want {
			t.Errorf("hasSELinuxContext(%q) = %v, want %v", tc.in, !tc.want, tc.want)
		}
	}
}
