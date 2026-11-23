package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSystemdPath_Rootful(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("test requires root privileges")
	}

	r := NewTargetDirResolver()
	dir, err := r.GetSystemdPath()
	if err != nil {
		t.Fatalf("GetSystemdPath() failed: %v", err)
	}

	expected := "/etc/containers/systemd"
	if dir != expected {
		t.Errorf("GetSystemdPath() = %q, want %q", dir, expected)
	}
}

func TestGetSystemdPath_Rootless(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	r := NewTargetDirResolver()

	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
		dir, err := r.GetSystemdPath()
		if err != nil {
			t.Fatalf("GetSystemdPath() failed: %v", err)
		}

		expected := "/tmp/test-config/containers/systemd"
		if dir != expected {
			t.Errorf("GetSystemdPath() = %q, want %q", dir, expected)
		}
	})

	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		dir, err := r.GetSystemdPath()
		if err != nil {
			t.Fatalf("GetSystemdPath() failed: %v", err)
		}

		if !filepath.IsAbs(dir) {
			t.Errorf("GetSystemdPath() returned non-absolute path: %q", dir)
		}
		if !filepath.IsAbs(dir) || filepath.Base(filepath.Dir(dir)) != "containers" {
			t.Errorf("GetSystemdPath() path doesn't contain containers/systemd: %q", dir)
		}
	})
}

func TestGetSystemdUnitPath_Rootful(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("test requires root privileges")
	}

	r := NewTargetDirResolver()
	dir, err := r.GetSystemdUnitPath()
	if err != nil {
		t.Fatalf("GetSystemdUnitPath() failed: %v", err)
	}

	expected := "/etc/systemd/system"
	if dir != expected {
		t.Errorf("GetSystemdUnitPath() = %q, want %q", dir, expected)
	}
}

func TestGetSystemdUnitPath_Rootless(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	r := NewTargetDirResolver()

	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
		dir, err := r.GetSystemdUnitPath()
		if err != nil {
			t.Fatalf("GetSystemdUnitPath() failed: %v", err)
		}

		expected := "/tmp/test-config/systemd/user"
		if dir != expected {
			t.Errorf("GetSystemdUnitPath() = %q, want %q", dir, expected)
		}
	})

	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		dir, err := r.GetSystemdUnitPath()
		if err != nil {
			t.Fatalf("GetSystemdUnitPath() failed: %v", err)
		}

		if !filepath.IsAbs(dir) {
			t.Errorf("GetSystemdUnitPath() returned non-absolute path: %q", dir)
		}
		if !filepath.IsAbs(dir) || filepath.Base(filepath.Dir(dir)) != "systemd" {
			t.Errorf("GetSystemdUnitPath() path doesn't contain systemd/user: %q", dir)
		}
	})
}

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
