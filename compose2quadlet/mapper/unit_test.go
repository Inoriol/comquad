package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestUnit_DependsOn(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DependsOn: types.DependsOnConfig{"db": {Condition: "service_started", Required: true}, "redis": {Condition: "service_started", Required: false}}}
	dirs := Unit(svc)

	assertDirective(t, dirs, "Requires", "db.container")
	assertDirective(t, dirs, "Wants", "redis.container")
	assertDirective(t, dirs, "After", "db.container")
	assertDirective(t, dirs, "After", "redis.container")
}

func TestUnit_DependsOnSortedOrder(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DependsOn: types.DependsOnConfig{
		"zdb": {Condition: "service_started", Required: true},
		"adb": {Condition: "service_started", Required: true},
		"mdb": {Condition: "service_started", Required: false},
	}}
	dirs := Unit(svc)

	var after []string
	for _, d := range dirs {
		if d.Key == "After" {
			after = append(after, d.Values...)
		}
	}
	want := []string{"adb.container", "mdb.container", "zdb.container"}
	if len(after) != len(want) {
		t.Fatalf("expected %d After values, got %v", len(want), after)
	}
	for i := range want {
		if after[i] != want[i] {
			t.Fatalf("expected sorted After %v, got %v", want, after)
		}
	}
}

func TestUnit_DependsOn_Restart(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DependsOn: types.DependsOnConfig{"db": {Condition: "service_started", Required: true, Restart: true}}}
	dirs := Unit(svc)

	assertDirective(t, dirs, "Requires", "db.container")
	assertDirective(t, dirs, "BindsTo", "db.container")
	assertDirective(t, dirs, "After", "db.container")
}

func TestUnit_DependsOn_Empty(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DependsOn: types.DependsOnConfig{}}
	dirs := Unit(svc)
	if len(dirs) != 0 {
		t.Fatalf("expected no directives, got %v", dirs)
	}
}

func TestUnit_DependsOn_ServiceHealthy(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DependsOn: types.DependsOnConfig{"db": {Condition: "service_healthy", Required: true}}}
	cfg := c2qtypes.DefaultConfig()
	serviceDirs := UnitService(svc, cfg)
	if len(serviceDirs) == 0 {
		t.Fatal("expected ExecStartPre for service_healthy dependency")
	}
	assertDirective(t, serviceDirs, "ExecStartPre", "/bin/sh -c 'while ! /usr/bin/podman healthcheck run db; do sleep 1; done'")
}

func TestUnit_DependsOn_ServiceCompletedSuccessfully(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DependsOn: types.DependsOnConfig{"init": {Condition: "service_completed_successfully", Required: true}}}
	unitDirs := Unit(svc)

	assertDirective(t, unitDirs, "Requires", "init.container")
	assertDirective(t, unitDirs, "After", "init.container")

	serviceDirs := UnitService(svc, c2qtypes.DefaultConfig())
	if len(serviceDirs) != 0 {
		t.Fatal("expected no health polling for service_completed_successfully")
	}
}

func TestUnit_DependsOn_MixedConditions(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DependsOn: types.DependsOnConfig{
		"db":    {Condition: "service_healthy", Required: true},
		"cache": {Condition: "service_started", Required: false},
	}}
	unitDirs := Unit(svc)
	assertDirective(t, unitDirs, "Requires", "db.container")
	assertDirective(t, unitDirs, "Wants", "cache.container")
	assertDirective(t, unitDirs, "After", "db.container")
	assertDirective(t, unitDirs, "After", "cache.container")

	serviceDirs := UnitService(svc, c2qtypes.DefaultConfig())
	if len(serviceDirs) != 1 {
		t.Fatalf("expected 1 ExecStartPre, got %d", len(serviceDirs))
	}
	assertDirective(t, serviceDirs, "ExecStartPre", "/bin/sh -c 'while ! /usr/bin/podman healthcheck run db; do sleep 1; done'")
}
