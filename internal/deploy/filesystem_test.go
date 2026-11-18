package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRuntimeSecretsDir_Rootful(t *testing.T) {
	// Skip if not running as root
	if os.Getuid() != 0 {
		t.Skip("test requires root privileges")
	}

	dir, err := ResolveRuntimeSecretsDir()
	if err != nil {
		t.Fatalf("ResolveRuntimeSecretsDir() failed: %v", err)
	}

	expected := "/run/comquad/secrets"
	if dir != expected {
		t.Errorf("ResolveRuntimeSecretsDir() = %q, want %q", dir, expected)
	}
}

func TestResolveRuntimeSecretsDir_Rootless(t *testing.T) {
	// Skip if running as root
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	// Test with XDG_RUNTIME_DIR set
	t.Setenv("XDG_RUNTIME_DIR", "/tmp/test-runtime")
	dir, err := ResolveRuntimeSecretsDir()
	if err != nil {
		t.Fatalf("ResolveRuntimeSecretsDir() failed: %v", err)
	}

	expected := "/tmp/test-runtime/comquad/secrets"
	if dir != expected {
		t.Errorf("ResolveRuntimeSecretsDir() = %q, want %q", dir, expected)
	}

	// Test without XDG_RUNTIME_DIR (should use /run/user/<uid>)
	t.Setenv("XDG_RUNTIME_DIR", "")
	dir, err = ResolveRuntimeSecretsDir()
	if err != nil {
		t.Fatalf("ResolveRuntimeSecretsDir() failed: %v", err)
	}

	// Verify it's an absolute path
	if dir == "" {
		t.Error("ResolveRuntimeSecretsDir() returned empty string")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("ResolveRuntimeSecretsDir() returned non-absolute path: %q", dir)
	}
	// Verify it contains comquad/secrets
	if !filepath.IsAbs(dir) || filepath.Base(filepath.Dir(dir)) != "comquad" {
		t.Errorf("ResolveRuntimeSecretsDir() path doesn't contain comquad/secrets: %q", dir)
	}
}
