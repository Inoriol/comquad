package preprocess

import (
	"testing"
)

func TestIsSELinuxEnabled_Override(t *testing.T) {
	trueVal := true
	SetSELinuxOverrides(&trueVal, "Enforcing")
	if !IsSELinuxEnabled() {
		t.Error("expected IsSELinuxEnabled to return true when overridden")
	}
}

func TestIsSELinuxEnabled_OverrideFalse(t *testing.T) {
	falseVal := false
	SetSELinuxOverrides(&falseVal, "Disabled")
	if IsSELinuxEnabled() {
		t.Error("expected IsSELinuxEnabled to return false when overridden")
	}
}

func TestSELinuxMode_Enforcing(t *testing.T) {
	SetSELinuxOverrides(nil, "Enforcing")
	if mode := SELinuxMode(); mode != "Enforcing" {
		t.Errorf("expected mode 'Enforcing', got %q", mode)
	}
}

func TestSELinuxMode_Permissive(t *testing.T) {
	SetSELinuxOverrides(nil, "Permissive")
	if mode := SELinuxMode(); mode != "Permissive" {
		t.Errorf("expected mode 'Permissive', got %q", mode)
	}
}

func TestSELinuxMode_Disabled(t *testing.T) {
	SetSELinuxOverrides(nil, "Disabled")
	if mode := SELinuxMode(); mode != "Disabled" {
		t.Errorf("expected mode 'Disabled', got %q", mode)
	}
}
