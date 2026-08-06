package graft

import (
	"testing"
)

func TestParsePullStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected PullStrategy
		hasError bool
	}{
		{"always", PullAlways, false},
		{"missing", PullMissing, false},
		{"never", PullNever, false},
		{"ALWAYS", PullAlways, false},
		{"Missing", PullMissing, false},
		{"Never", PullNever, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		result, err := ParsePullStrategy(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParsePullStrategy(%q) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParsePullStrategy(%q) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParsePullStrategy(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	}
}
