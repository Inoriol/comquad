package build

import (
	"testing"
)

func TestGenerateBuildTag(t *testing.T) {
	tests := []struct {
		project   string
		service   string
		expected  string
	}{
		{"myapp", "web", "myapp-web:latest"},
		{"test-project", "db", "test-project-db:latest"},
		{"proj", "api", "proj-api:latest"},
	}

	for _, tt := range tests {
		result := GenerateBuildTag(tt.project, tt.service)
		if result != tt.expected {
			t.Errorf("GenerateBuildTag(%q, %q) = %q, want %q", tt.project, tt.service, result, tt.expected)
		}
	}
}

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

func TestGetBuildArgs(t *testing.T) {
	args := map[string]string{
		"VERSION": "1.0",
		"ARCH":    "amd64",
	}

	result := GetBuildArgs(args)

	if len(result) != 2 {
		t.Errorf("GetBuildArgs returned %d args, want 2", len(result))
	}

	// Check that args are in the correct format
	found := make(map[string]bool)
	for _, arg := range result {
		if arg == "VERSION=1.0" {
			found["VERSION"] = true
		}
		if arg == "ARCH=amd64" {
			found["ARCH"] = true
		}
	}

	if !found["VERSION"] {
		t.Error("Expected VERSION=1.0 in build args")
	}
	if !found["ARCH"] {
		t.Error("Expected ARCH=amd64 in build args")
	}
}

func TestGetBuildArgsEmpty(t *testing.T) {
	result := GetBuildArgs(map[string]string{})
	if len(result) != 0 {
		t.Errorf("GetBuildArgs(empty) returned %d args, want 0", len(result))
	}
}
