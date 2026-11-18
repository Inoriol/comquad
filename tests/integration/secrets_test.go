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

	// Verify secret is accessible inside container (Podman secrets use full prefixed name)
	secretPath := fmt.Sprintf("/run/secrets/cq-%s-db_password", project)
	helpers.AssertSecretExists(t, containerName, secretPath)
	helpers.AssertSecretContent(t, containerName, secretPath, "super-secret-password")

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

	// Verify Podman secret exists
	helpers.AssertPodmanSecretExists(t, fmt.Sprintf("cq-%s-db_password", project))

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)

	// Verify secret service is gone
	helpers.AssertSecretServiceGone(t, project, "db_password", false)
	helpers.AssertSecretQuadletFileGone(t, project, "db_password")
	helpers.AssertProjectGone(t, project)

	// Verify Podman secret is removed
	helpers.AssertPodmanSecretGone(t, fmt.Sprintf("cq-%s-db_password", project))
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

	// Verify secret is accessible inside container (Podman secrets use full prefixed name)
	secretPath := fmt.Sprintf("/run/secrets/cq-%s-api_key", project)
	helpers.AssertSecretExists(t, containerName, secretPath)
	helpers.AssertSecretContent(t, containerName, secretPath, secretContent)

	// Verify quadlet file exists
	helpers.AssertSecretQuadletFileExists(t, project, "api_key")

	// Verify Podman secret exists
	helpers.AssertPodmanSecretExists(t, fmt.Sprintf("cq-%s-api_key", project))

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)

	// Verify cleanup
	helpers.AssertSecretServiceGone(t, project, "api_key", false)
	helpers.AssertSecretQuadletFileGone(t, project, "api_key")

	// Verify Podman secret is removed
	helpers.AssertPodmanSecretGone(t, fmt.Sprintf("cq-%s-api_key", project))
}

func TestSecret_CustomPermissions(t *testing.T) {
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

	// Verify secret service is active (custom permissions now use native Secret= directive)
	helpers.AssertSecretServiceActive(t, project, "custom_secret", false)

	// Verify container is running
	containerName := fmt.Sprintf("%s-web", project)
	helpers.AssertContainerRunning(t, containerName)

	// Verify secret is accessible at custom target path
	helpers.AssertSecretExists(t, containerName, "/run/secrets/custom_secret")
	helpers.AssertSecretContent(t, containerName, "/run/secrets/custom_secret", secretContent)

	// Verify .cq-secret file exists (no longer falls back to volume mount for Podman 4.5.0+)
	helpers.AssertSecretQuadletFileExists(t, project, "custom_secret")

	// Verify Podman secret exists
	helpers.AssertPodmanSecretExists(t, fmt.Sprintf("cq-%s-custom_secret", project))

	// Verify the secret is mounted via Podman secret (not direct bind mount)
	mounts := helpers.GetContainerMounts(t, containerName)
	found := false
	for _, m := range mounts {
		if m.Destination == "/run/secrets/custom_secret" {
			found = true
			// Should be from Podman secret store, not a direct bind mount
			if !strings.Contains(m.Source, "secrets") {
				t.Fatalf("expected Podman secret mount, but got: %s", m.Source)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected mount at /run/secrets/custom_secret")
	}

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)
	helpers.AssertSecretServiceGone(t, project, "custom_secret", false)
	helpers.AssertSecretQuadletFileGone(t, project, "custom_secret")
	helpers.AssertPodmanSecretGone(t, fmt.Sprintf("cq-%s-custom_secret", project))
	helpers.AssertProjectGone(t, project)
}

func TestSecret_EnvironmentVariableExposure(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)

	compose := `name: ` + project + `
services:
  web:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    secrets:
      - source: db_password
        x-type: env
        x-target: DB_PASSWORD
secrets:
  db_password:
    environment: DB_PASSWORD_ENV
`
	dir, _ := helpers.WriteCompose(t, compose)

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	t.Setenv("DB_PASSWORD_ENV", "secret-env-value")

	// Deploy
	helpers.MustSucceed(t, dir, "up", "--name", project)

	// Verify secret service is active
	helpers.AssertSecretServiceActive(t, project, "db_password", false)

	// Verify container is running
	containerName := fmt.Sprintf("%s-web", project)
	helpers.AssertContainerRunning(t, containerName)

	// Verify secret is accessible as environment variable
	envValue := helpers.ExecInContainer(t, containerName, "printenv DB_PASSWORD")
	if envValue != "secret-env-value" {
		t.Fatalf("expected DB_PASSWORD env var to be 'secret-env-value', got %q", envValue)
	}

	// Verify Podman secret exists
	helpers.AssertPodmanSecretExists(t, fmt.Sprintf("cq-%s-db_password", project))

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)
	helpers.AssertSecretServiceGone(t, project, "db_password", false)
	helpers.AssertPodmanSecretGone(t, fmt.Sprintf("cq-%s-db_password", project))
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

	// Verify both containers can access the secret (Podman secrets use full prefixed name)
	secretPath := fmt.Sprintf("/run/secrets/cq-%s-shared_secret", project)
	helpers.AssertSecretContent(t, webContainer, secretPath, "shared-value-123")
	helpers.AssertSecretContent(t, apiContainer, secretPath, "shared-value-123")

	// Verify state tracks only one secret
	state := helpers.AssertProjectRegistered(t, project)
	if len(state.Resources.Secrets) != 1 {
		t.Fatalf("expected exactly 1 secret in state (deduplicated), got %d: %v",
			len(state.Resources.Secrets), state.Resources.Secrets)
	}

	// Verify Podman secret exists
	helpers.AssertPodmanSecretExists(t, fmt.Sprintf("cq-%s-shared_secret", project))

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)
	helpers.AssertSecretServiceGone(t, project, "shared_secret", false)

	// Verify Podman secret is removed
	helpers.AssertPodmanSecretGone(t, fmt.Sprintf("cq-%s-shared_secret", project))
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

	// Verify both secrets are accessible (Podman secrets use full prefixed name)
	secretOnePath := fmt.Sprintf("/run/secrets/cq-%s-secret_one", project)
	secretTwoPath := fmt.Sprintf("/run/secrets/cq-%s-secret_two", project)
	helpers.AssertSecretContent(t, containerName, secretOnePath, "value-one")
	helpers.AssertSecretContent(t, containerName, secretTwoPath, "value-two")

	// Verify state tracks both secrets
	state := helpers.AssertProjectRegistered(t, project)
	if len(state.Resources.Secrets) != 2 {
		t.Fatalf("expected 2 secrets in state, got %d: %v",
			len(state.Resources.Secrets), state.Resources.Secrets)
	}

	// Verify Podman secrets exist
	helpers.AssertPodmanSecretExists(t, fmt.Sprintf("cq-%s-secret_one", project))
	helpers.AssertPodmanSecretExists(t, fmt.Sprintf("cq-%s-secret_two", project))

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)
	helpers.AssertSecretServiceGone(t, project, "secret_one", false)
	helpers.AssertSecretServiceGone(t, project, "secret_two", false)

	// Verify Podman secrets are removed
	helpers.AssertPodmanSecretGone(t, fmt.Sprintf("cq-%s-secret_one", project))
	helpers.AssertPodmanSecretGone(t, fmt.Sprintf("cq-%s-secret_two", project))
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

func TestSecret_DriverWithFile(t *testing.T) {
	helpers.SkipIfSystemdUnavailable(t)
	project := helpers.ProjectName(t)

	compose := `name: ` + project + `
services:
  web:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    secrets:
      - source: file_secret
secrets:
  file_secret:
    file: /tmp/test-driver-secret.txt
    driver: file
`
	dir, _ := helpers.WriteCompose(t, compose)

	t.Cleanup(func() {
		helpers.Comquad(t, dir, "down", "--name", project)
	})

	// Create the secret file
	secretContent := "driver-secret-value"
	if err := os.WriteFile("/tmp/test-driver-secret.txt", []byte(secretContent), 0600); err != nil {
		t.Fatalf("failed to create test secret file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove("/tmp/test-driver-secret.txt")
	})

	// Deploy
	helpers.MustSucceed(t, dir, "up", "--name", project)

	// Verify secret service is active
	helpers.AssertSecretServiceActive(t, project, "file_secret", false)

	// Verify container is running
	containerName := fmt.Sprintf("%s-web", project)
	helpers.AssertContainerRunning(t, containerName)

	// Verify secret is accessible inside container
	secretPath := fmt.Sprintf("/run/secrets/cq-%s-file_secret", project)
	helpers.AssertSecretExists(t, containerName, secretPath)
	helpers.AssertSecretContent(t, containerName, secretPath, secretContent)

	// Verify Podman secret exists
	helpers.AssertPodmanSecretExists(t, fmt.Sprintf("cq-%s-file_secret", project))

	// Verify the secret was created with the file driver by inspecting it
	secretInfo := helpers.GetPodmanSecretInfo(t, fmt.Sprintf("cq-%s-file_secret", project))
	if secretInfo.SecretDriver != "file" {
		t.Fatalf("expected secret driver to be 'file', got %q", secretInfo.SecretDriver)
	}

	// Tear down
	helpers.MustSucceed(t, dir, "down", "--name", project)
	helpers.AssertSecretServiceGone(t, project, "file_secret", false)
	helpers.AssertPodmanSecretGone(t, fmt.Sprintf("cq-%s-file_secret", project))
}
