//go:build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/tests/integration/helpers"
)

func TestSecret_EnvironmentTmpfs(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SecretEnvCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Set the environment variable before deploying
	t.Setenv("DB_PASSWORD", "super-secret-password")

	// Deploy
	helpers.MustSucceed(t, dir, "up", "--name", project)

	// Verify secret service is active
	helpers.AssertSecretServiceActive(t, project, "db_password", false)

	// Verify container is running
	containerName := fmt.Sprintf("%s-web", project)
	helpers.AssertContainerRunning(t, containerName)

	// Verify secret is accessible inside container
	helpers.AssertSecretExists(t, containerName, "/run/secrets/db_password")
	helpers.AssertSecretContent(t, containerName, "/run/secrets/db_password", "super-secret-password")

	// Verify quadlet file exists
	helpers.AssertSecretQuadletFileExists(t, project, "db_password")

	// Verify state tracks the secret
	state := helpers.AssertProjectRegistered(t, project)
	if len(state.Resources.Secrets) == 0 {
		t.Fatal("expected state.Resources.Secrets to be non-empty")
	}
	found := false
	for _, s := range state.Resources.Secrets {
		if s == "db_password" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected db_password in state.Resources.Secrets, got %v", state.Resources.Secrets)
	}

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)

	// Verify secret service is gone
	helpers.AssertSecretServiceGone(t, project, "db_password", false)
	helpers.AssertSecretQuadletFileGone(t, project, "db_password")
	helpers.AssertProjectGone(t, project)
}

func TestSecret_FileTmpfs(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SecretFileCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Create the secret file
	secretContent := "my-api-key-12345"
	if err := os.WriteFile("/tmp/test-api.key", []byte(secretContent), 0600); err != nil {
		t.Fatalf("failed to create test secret file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove("/tmp/test-api.key")
	})

	// Deploy
	helpers.MustSucceed(t, dir, "up", "--name", project)

	// Verify secret service is active
	helpers.AssertSecretServiceActive(t, project, "api_key", false)

	// Verify container is running
	containerName := fmt.Sprintf("%s-web", project)
	helpers.AssertContainerRunning(t, containerName)

	// Verify secret is accessible inside container
	helpers.AssertSecretExists(t, containerName, "/run/secrets/api_key")
	helpers.AssertSecretContent(t, containerName, "/run/secrets/api_key", secretContent)

	// Verify quadlet file exists
	helpers.AssertSecretQuadletFileExists(t, project, "api_key")

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)

	// Verify cleanup
	helpers.AssertSecretServiceGone(t, project, "api_key", false)
	helpers.AssertSecretQuadletFileGone(t, project, "api_key")
}

func TestSecret_CustomPermissionsFallback(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SecretCustomPermsCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Create the secret file
	secretContent := "custom-secret-value"
	if err := os.WriteFile("/tmp/test-custom.key", []byte(secretContent), 0600); err != nil {
		t.Fatalf("failed to create test secret file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove("/tmp/test-custom.key")
	})

	// Deploy
	helpers.MustSucceed(t, dir, "up", "--name", project)

	// Verify container is running
	containerName := fmt.Sprintf("%s-web", project)
	helpers.AssertContainerRunning(t, containerName)

	// Verify secret is accessible at custom target path
	helpers.AssertSecretExists(t, containerName, "/run/secrets/custom_secret")
	helpers.AssertSecretContent(t, containerName, "/run/secrets/custom_secret", secretContent)

	// Verify NO .cq-secret file exists (fallback to volume mount)
	basename := "cq-" + project + "-custom_secret.cq-secret"
	if helpers.QuadletFileExists(t, basename) {
		t.Fatalf("expected no .cq-secret file for custom permissions fallback, but %s exists", basename)
	}

	// Verify the mount is a direct bind mount (not from /run/credentials)
	mounts := helpers.GetContainerMounts(t, containerName)
	found := false
	for _, m := range mounts {
		if m.Destination == "/run/secrets/custom_secret" {
			found = true
			// Should be a bind mount from the actual file, not from /run/credentials
			if strings.Contains(m.Source, "/run/credentials") {
				t.Fatalf("expected direct bind mount, but got /run/credentials path: %s", m.Source)
			}
			if !strings.Contains(m.Mode, "ro") {
				t.Fatalf("expected read-only mount, but got mode: %s", m.Mode)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected mount at /run/secrets/custom_secret")
	}

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)
	helpers.AssertProjectGone(t, project)
}

func TestSecret_Deduplication(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)
	dir, _ := helpers.WriteCompose(t, helpers.SecretMultiCompose(project))

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Set the environment variable
	t.Setenv("SHARED_SECRET", "shared-value-123")

	// Deploy
	helpers.MustSucceed(t, dir, "up", "--name", project)

	// Verify only ONE secret service exists (deduplicated)
	helpers.AssertSecretServiceActive(t, project, "shared_secret", false)

	// Verify both containers are running
	webContainer := fmt.Sprintf("%s-web", project)
	apiContainer := fmt.Sprintf("%s-api", project)
	helpers.AssertContainerRunning(t, webContainer)
	helpers.AssertContainerRunning(t, apiContainer)

	// Verify both containers can access the secret
	helpers.AssertSecretContent(t, webContainer, "/run/secrets/shared_secret", "shared-value-123")
	helpers.AssertSecretContent(t, apiContainer, "/run/secrets/shared_secret", "shared-value-123")

	// Verify state tracks only one secret
	state := helpers.AssertProjectRegistered(t, project)
	if len(state.Resources.Secrets) != 1 {
		t.Fatalf("expected exactly 1 secret in state (deduplicated), got %d: %v",
			len(state.Resources.Secrets), state.Resources.Secrets)
	}

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)
	helpers.AssertSecretServiceGone(t, project, "shared_secret", false)
}

func TestSecret_MultipleSecretsPerService(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)

	compose := `name: ` + project + `
services:
  web:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    secrets:
      - secret_one
      - secret_two
secrets:
  secret_one:
    environment: SECRET_ONE
  secret_two:
    file: /tmp/test-secret-two.txt
`
	dir, _ := helpers.WriteCompose(t, compose)

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Set up secrets
	t.Setenv("SECRET_ONE", "value-one")
	if err := os.WriteFile("/tmp/test-secret-two.txt", []byte("value-two"), 0600); err != nil {
		t.Fatalf("failed to create test secret file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove("/tmp/test-secret-two.txt")
	})

	// Deploy
	helpers.MustSucceed(t, dir, "up", "--name", project)

	// Verify both secret services are active
	helpers.AssertSecretServiceActive(t, project, "secret_one", false)
	helpers.AssertSecretServiceActive(t, project, "secret_two", false)

	// Verify container is running
	containerName := fmt.Sprintf("%s-web", project)
	helpers.AssertContainerRunning(t, containerName)

	// Verify both secrets are accessible
	helpers.AssertSecretContent(t, containerName, "/run/secrets/secret_one", "value-one")
	helpers.AssertSecretContent(t, containerName, "/run/secrets/secret_two", "value-two")

	// Verify state tracks both secrets
	state := helpers.AssertProjectRegistered(t, project)
	if len(state.Resources.Secrets) != 2 {
		t.Fatalf("expected 2 secrets in state, got %d: %v",
			len(state.Resources.Secrets), state.Resources.Secrets)
	}

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)
	helpers.AssertSecretServiceGone(t, project, "secret_one", false)
	helpers.AssertSecretServiceGone(t, project, "secret_two", false)
}

func TestSecret_CustomTargetPath(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)

	compose := `name: ` + project + `
services:
  web:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    secrets:
      - source: my_secret
        target: /app/config/secret.txt
secrets:
  my_secret:
    environment: MY_SECRET
`
	dir, _ := helpers.WriteCompose(t, compose)

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	t.Setenv("MY_SECRET", "custom-target-value")

	// Deploy
	helpers.MustSucceed(t, dir, "up", "--name", project)

	// Verify secret is at custom target path
	containerName := fmt.Sprintf("%s-web", project)
	helpers.AssertContainerRunning(t, containerName)
	helpers.AssertSecretContent(t, containerName, "/app/config/secret.txt", "custom-target-value")

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)
}
