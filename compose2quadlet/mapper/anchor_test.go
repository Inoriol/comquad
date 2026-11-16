package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestSelectAnchor_NoDependencies(t *testing.T) {
	services := map[string]types.ServiceConfig{
		"web":   {Name: "web"},
		"db":    {Name: "db"},
		"cache": {Name: "cache"},
	}

	anchor := SelectAnchor(services)
	if anchor != "cache" {
		t.Errorf("expected alphabetically first service 'cache', got %q", anchor)
	}
}

func TestSelectAnchor_WithDependencies(t *testing.T) {
	services := map[string]types.ServiceConfig{
		"web": {Name: "web", DependsOn: types.DependsOnConfig{
			"db": {Condition: "service_started"},
		}},
		"db":    {Name: "db"},
		"cache": {Name: "cache"},
	}

	anchor := SelectAnchor(services)
	if anchor != "cache" {
		t.Errorf("expected root node 'cache' (alphabetically first root), got %q", anchor)
	}
}

func TestSelectAnchor_SingleRoot(t *testing.T) {
	services := map[string]types.ServiceConfig{
		"web": {Name: "web", DependsOn: types.DependsOnConfig{
			"db": {Condition: "service_started"},
		}},
		"db": {Name: "db", DependsOn: types.DependsOnConfig{
			"init": {Condition: "service_completed_successfully"},
		}},
		"init": {Name: "init"},
	}

	anchor := SelectAnchor(services)
	if anchor != "init" {
		t.Errorf("expected root node 'init', got %q", anchor)
	}
}

func TestSelectAnchor_MultipleRoots(t *testing.T) {
	services := map[string]types.ServiceConfig{
		"web": {Name: "web", DependsOn: types.DependsOnConfig{
			"db": {Condition: "service_started"},
		}},
		"db":    {Name: "db"},
		"redis": {Name: "redis"},
	}

	anchor := SelectAnchor(services)
	if anchor != "db" {
		t.Errorf("expected alphabetically first root 'db', got %q", anchor)
	}
}

func TestSelectAnchor_EmptyServices(t *testing.T) {
	services := map[string]types.ServiceConfig{}

	anchor := SelectAnchor(services)
	if anchor != "" {
		t.Errorf("expected empty string for empty services, got %q", anchor)
	}
}

func TestSelectAnchor_SingleService(t *testing.T) {
	services := map[string]types.ServiceConfig{
		"web": {Name: "web"},
	}

	anchor := SelectAnchor(services)
	if anchor != "web" {
		t.Errorf("expected 'web', got %q", anchor)
	}
}
