package compose2quadlet_test

import (
	"context"
	"testing"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	c2q "github.com/Inoriol/comquad/compose2quadlet"
)

func loadProject(t *testing.T, yamlPath string) *types.Project {
	t.Helper()
	opts, err := cli.NewProjectOptions(
		[]string{yamlPath},
		cli.WithOsEnv,
		cli.WithDotEnv,
	)
	if err != nil {
		t.Fatalf("failed to create project options from %s: %v", yamlPath, err)
	}
	project, err := opts.LoadProject(context.Background())
	if err != nil {
		t.Fatalf("failed to load project from %s: %v", yamlPath, err)
	}
	return project
}

func hasSection(unit c2q.QuadletUnit, sectionName string) (c2q.Section, bool) {
	for _, s := range unit.Sections {
		if s.Name == sectionName {
			return s, true
		}
	}
	return c2q.Section{}, false
}

func findUnit(units []c2q.QuadletUnit, name string, unitType c2q.UnitType) (c2q.QuadletUnit, bool) {
	for _, u := range units {
		if u.Name == name && u.Type == unitType {
			return u, true
		}
	}
	return c2q.QuadletUnit{}, false
}

func directiveValue(dirs []c2q.Directive, key string) (string, bool) {
	for _, d := range dirs {
		if d.Key == key && len(d.Values) > 0 {
			return d.Values[0], true
		}
	}
	return "", false
}

func hasDirectiveValue(dirs []c2q.Directive, key, value string) bool {
	for _, d := range dirs {
		if d.Key != key {
			continue
		}
		if len(d.Values) == 0 && value == "" {
			return true
		}
		for _, v := range d.Values {
			if v == value {
				return true
			}
		}
	}
	return false
}
