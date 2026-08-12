package mapper

import (
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func assertDirective(t *testing.T, dirs []c2qtypes.Directive, key, value string) {
	t.Helper()
	for _, d := range dirs {
		if d.Key != key {
			continue
		}
		if len(d.Values) == 0 && value == "" {
			return
		}
		for _, v := range d.Values {
			if v == value {
				return
			}
		}
	}
	t.Fatalf("directive %s=%s not found in %v", key, value, dirs)
}

func assertNoDirective(t *testing.T, dirs []c2qtypes.Directive, key string) {
	t.Helper()
	for _, d := range dirs {
		if d.Key == key {
			t.Fatalf("unexpected directive %s found in %v", key, dirs)
		}
	}
}

func hasDirective(dirs []c2qtypes.Directive, key, value string) bool {
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

func hasWarning(cfg *c2qtypes.Config, service, field string) bool {
	for _, w := range cfg.Warnings {
		if w.Service == service && (w.Field == field || strings.Contains(w.Field, field)) {
			return true
		}
	}
	return false
}

func boolPtr(b bool) *bool                          { return &b }
func strPtr(s string) *string                       { return &s }
func uintPtr(u uint64) *uint64                      { return &u }
func durationPtr(d types.Duration) *types.Duration   { return &d }
