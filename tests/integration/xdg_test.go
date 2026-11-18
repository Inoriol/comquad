//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/tests/integration/helpers"
)

// TestXDG_StateDirectory verifies that project state is stored in XDG_STATE_HOME.
func TestXDG_StateDirectory(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)

	projectName := helpers.ProjectName(t)
	composeContent := helpers.SimpleCompose(projectName)
	projectDir, _ := helpers.WriteCompose(t, composeContent)

	// Deploy the project
	helpers.MustSucceed(t, projectDir, "up", "--no-diff", "-n", projectName)
	defer helpers.MustSucceed(t, projectDir, "down", "-y", "-n", projectName)

	// Verify state file is in XDG_STATE_HOME
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("could not determine home dir: %v", err)
		}
		stateDir = filepath.Join(home, ".local", "state")
	}

	statePath := filepath.Join(stateDir, "comquad", "projects.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Errorf("state file not found at expected XDG_STATE_HOME location: %s", statePath)
	}

	// Verify baseline is in XDG_STATE_HOME
	baselineDir := filepath.Join(stateDir, "comquad", "baseline", projectName)
	if _, err := os.Stat(baselineDir); os.IsNotExist(err) {
		t.Errorf("baseline directory not found at expected XDG_STATE_HOME location: %s", baselineDir)
	}

	// Verify project is registered
	helpers.AssertProjectRegistered(t, projectName)
}

// TestXDG_RuntimeSecretsDirectory verifies that runtime secrets use XDG_RUNTIME_DIR.
func TestXDG_RuntimeSecretsDirectory(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)

	projectName := helpers.ProjectName(t)
	composeContent := helpers.SecretEnvCompose(projectName)
	projectDir, _ := helpers.WriteCompose(t, composeContent)

	// Set environment variable for the secret
	t.Setenv("DB_PASSWORD", "test-secret-value")

	// Deploy the project
	helpers.MustSucceed(t, projectDir, "up", "--no-diff", "-n", projectName)
	defer helpers.MustSucceed(t, projectDir, "down", "-y", "-n", projectName)

	// Verify runtime secrets directory
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		// In rootless mode, XDG_RUNTIME_DIR should be set
		t.Skip("XDG_RUNTIME_DIR not set, skipping runtime secrets test")
	}

	secretsDir := filepath.Join(runtimeDir, "comquad", "secrets")
	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("could not read runtime secrets directory %s: %v", secretsDir, err)
	}

	// Find the secret file for our project
	found := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "cq-"+projectName+"-") && strings.HasSuffix(entry.Name(), ".env") {
			found = true
			// Verify the secret file contains the expected value
			content, err := os.ReadFile(filepath.Join(secretsDir, entry.Name()))
			if err != nil {
				t.Errorf("could not read secret file %s: %v", entry.Name(), err)
			}
			if string(content) != "test-secret-value" {
				t.Errorf("secret file %s has unexpected content: got %q, want %q", entry.Name(), string(content), "test-secret-value")
			}
			break
		}
	}
	if !found {
		t.Errorf("runtime secret file not found in %s", secretsDir)
	}
}

// TestXDG_CacheDirectory verifies that build cache uses XDG_CACHE_HOME.
func TestXDG_CacheDirectory(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)

	projectName := helpers.ProjectName(t)
	composeContent := `name: ` + projectName + `
services:
  app:
    build: .
`
	projectDir, _ := helpers.WriteCompose(t, composeContent)

	// Create a simple Dockerfile
	dockerfileContent := `FROM docker.io/library/alpine:latest
CMD ["echo", "hello"]
`
	helpers.WriteFile(t, projectDir, "Dockerfile", dockerfileContent)

	// Build the project
	helpers.MustSucceed(t, projectDir, "build", "--no-diff", "-n", projectName)
	defer helpers.MustSucceed(t, projectDir, "down", "-y", "-n", projectName)

	// Verify cache directory
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("could not determine home dir: %v", err)
		}
		cacheDir = filepath.Join(home, ".cache")
	}

	buildCacheDir := filepath.Join(cacheDir, "comquad", "builds", projectName)
	if _, err := os.Stat(buildCacheDir); os.IsNotExist(err) {
		t.Errorf("build cache directory not found at expected XDG_CACHE_HOME location: %s", buildCacheDir)
	}
}

// TestXDG_CustomDirectories verifies that custom XDG directories are respected.
func TestXDG_CustomDirectories(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)

	// Create temporary directories for custom XDG paths
	tmpDir := t.TempDir()
	customStateDir := filepath.Join(tmpDir, "state")
	customCacheDir := filepath.Join(tmpDir, "cache")

	if err := os.MkdirAll(customStateDir, 0755); err != nil {
		t.Fatalf("could not create custom state dir: %v", err)
	}
	if err := os.MkdirAll(customCacheDir, 0755); err != nil {
		t.Fatalf("could not create custom cache dir: %v", err)
	}

	projectName := helpers.ProjectName(t)
	composeContent := helpers.SimpleCompose(projectName)
	projectDir, _ := helpers.WriteCompose(t, composeContent)

	// Deploy with custom XDG directories
	env := map[string]string{
		"XDG_STATE_HOME": customStateDir,
		"XDG_CACHE_HOME": customCacheDir,
	}
	helpers.ComquadWithEnv(t, projectDir, env, "up", "--no-diff", "-n", projectName)
	defer helpers.ComquadWithEnv(t, projectDir, env, "down", "-y", "-n", projectName)

	// Verify state is in custom directory
	statePath := filepath.Join(customStateDir, "comquad", "projects.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Errorf("state file not found at custom XDG_STATE_HOME location: %s", statePath)
	}
}
