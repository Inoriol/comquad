package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/internal/preprocess"
)

func TestSecretHandler_ExternalSecret(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\nPublishPort=80:80\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"apikey": {Name: "apikey", External: true},
	}
	refs := preprocess.ServiceSecretRefs{
		"web": {{Source: "apikey"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, t.TempDir(), false, false)

	content := result[containerPath]
	if !strings.Contains(content, "Secret=apikey") {
		t.Errorf("expected Secret=apikey in container, got:\n%s", content)
	}
	diskContent := readFile(t, containerPath)
	if !strings.Contains(diskContent, "Secret=apikey") {
		t.Errorf("expected Secret=apikey on disk, got:\n%s", diskContent)
	}
}

func TestSecretHandler_ExternalWithName(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"cert": {Name: "cert", External: true, ExternalName: "CERT_KEY"},
	}
	refs := preprocess.ServiceSecretRefs{
		"web": {{Source: "cert"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, t.TempDir(), false, false)

	content := result[containerPath]
	if !strings.Contains(content, "Secret=CERT_KEY") {
		t.Errorf("expected Secret=CERT_KEY, got:\n%s", content)
	}
}

func TestSecretHandler_FileSecret(t *testing.T) {
	dir := t.TempDir()

	secretFile := filepath.Join(dir, "secret.txt")
	os.WriteFile(secretFile, []byte("supersecret"), 0600)

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"db_password": {Name: "db_password", File: secretFile},
	}
	refs := preprocess.ServiceSecretRefs{
		"web": {{Source: "db_password"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, t.TempDir(), false, false)

	content := result[containerPath]
	expectedVolume := "Volume=" + secretFile + ":/run/secrets/db_password:ro"
	if !strings.Contains(content, expectedVolume) {
		t.Errorf("expected %q, got:\n%s", expectedVolume, content)
	}
	if strings.Contains(content, "[Service]") {
		t.Error("expected no [Service] section for direct bind mount")
	}
}

func TestSecretHandler_EnvironmentSecret(t *testing.T) {
	dir := t.TempDir()
	secretsDir := t.TempDir()

	os.Setenv("TEST_TOKEN", "secret-token-value")
	defer os.Unsetenv("TEST_TOKEN")

	containerPath := writeTestFile(t, dir, "cq-myapp-api.container",
		"[Container]\nImage=docker.io/library/alpine:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"token": {Name: "token", Environment: "TEST_TOKEN"},
	}
	refs := preprocess.ServiceSecretRefs{
		"api": {{Source: "token"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, secretsDir, false, false)

	content := result[containerPath]
	expectedVolume := "Volume=" + filepath.Join(secretsDir, "token") + ":/run/secrets/token:ro"
	if !strings.Contains(content, expectedVolume) {
		t.Errorf("expected %q, got:\n%s", expectedVolume, content)
	}

	secretFile := filepath.Join(secretsDir, "token")
	writtenContent, err := os.ReadFile(secretFile)
	if err != nil {
		t.Fatalf("failed to read managed secret file: %v", err)
	}
	if string(writtenContent) != "secret-token-value" {
		t.Errorf("expected secret content 'secret-token-value', got %q", string(writtenContent))
	}
}

func TestSecretHandler_EnvironmentSecretDryRun(t *testing.T) {
	dir := t.TempDir()
	secretsDir := t.TempDir()

	os.Setenv("TEST_TOKEN", "secret-token-value")
	defer os.Unsetenv("TEST_TOKEN")

	containerPath := writeTestFile(t, dir, "cq-myapp-api.container",
		"[Container]\nImage=docker.io/library/alpine:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"token": {Name: "token", Environment: "TEST_TOKEN"},
	}
	refs := preprocess.ServiceSecretRefs{
		"api": {{Source: "token"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, secretsDir, true, false)

	content := result[containerPath]
	if !strings.Contains(content, "Volume=") {
		t.Errorf("expected Volume= directive in dry run, got:\n%s", content)
	}

	secretFile := filepath.Join(secretsDir, "token")
	if _, err := os.Stat(secretFile); !os.IsNotExist(err) {
		t.Error("expected no secret file on disk in dry run")
	}
}

func TestSecretHandler_MultipleSecrets(t *testing.T) {
	dir := t.TempDir()

	secretFile := filepath.Join(dir, "secret.txt")
	os.WriteFile(secretFile, []byte("supersecret"), 0600)

	os.Setenv("TEST_TOKEN", "tokenval")
	defer os.Unsetenv("TEST_TOKEN")

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"db_pass":   {Name: "db_pass", File: secretFile},
		"token":     {Name: "token", Environment: "TEST_TOKEN"},
		"ext_key":   {Name: "ext_key", External: true},
		"named_ext": {Name: "named_ext", External: true, ExternalName: "ALT_KEY"},
	}
	refs := preprocess.ServiceSecretRefs{
		"web": {
			{Source: "db_pass"},
			{Source: "token"},
			{Source: "ext_key"},
			{Source: "named_ext"},
		},
	}

	result := SecretHandler(files, "myapp", defs, refs, t.TempDir(), false, false)

	content := result[containerPath]

	checks := []string{
		"Secret=ext_key",
		"Secret=ALT_KEY",
		"Volume=" + secretFile + ":/run/secrets/db_pass:ro",
		"Volume=",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected %q in output, got:\n%s", check, content)
		}
	}

	if strings.Contains(content, "[Service]") {
		t.Error("expected no [Service] section")
	}
}

func TestSecretHandler_CustomTarget(t *testing.T) {
	dir := t.TempDir()

	secretFile := filepath.Join(dir, "secret.txt")
	os.WriteFile(secretFile, []byte("supersecret"), 0600)

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"db_password": {Name: "db_password", File: secretFile},
	}
	refs := preprocess.ServiceSecretRefs{
		"web": {{Source: "db_password", Target: "/var/lib/secrets/mypass"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, t.TempDir(), false, false)

	content := result[containerPath]
	expectedVolume := "Volume=" + secretFile + ":/var/lib/secrets/mypass:ro"
	if !strings.Contains(content, expectedVolume) {
		t.Errorf("expected custom target %q, got:\n%s", expectedVolume, content)
	}
}

func TestSecretHandler_SkipsNonContainer(t *testing.T) {
	dir := t.TempDir()

	netPath := writeTestFile(t, dir, "cq-myapp-default.network", "[Network]\n")

	files := map[string]string{
		netPath: readFile(t, netPath),
	}

	defs := map[string]preprocess.SecretDef{
		"apikey": {Name: "apikey", External: true},
	}
	refs := preprocess.ServiceSecretRefs{
		"web": {{Source: "apikey"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, t.TempDir(), false, false)

	if len(result) != 1 {
		t.Errorf("expected 1 file in result, got %d", len(result))
	}
	content := result[netPath]
	if strings.Contains(content, "Secret=") {
		t.Error("expected no Secret= in network file")
	}
}

func TestSecretHandler_NoSecretsForService(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"apikey": {Name: "apikey", External: true},
	}
	refs := preprocess.ServiceSecretRefs{}

	result := SecretHandler(files, "myapp", defs, refs, t.TempDir(), false, false)

	content := result[containerPath]
	if strings.Contains(content, "Secret=") {
		t.Error("expected no Secret= for service without refs")
	}
}

func TestSecretHandler_FileSecretWithSELinux(t *testing.T) {
	dir := t.TempDir()

	secretFile := filepath.Join(dir, "secret.txt")
	os.WriteFile(secretFile, []byte("supersecret"), 0600)

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"db_password": {Name: "db_password", File: secretFile},
	}
	refs := preprocess.ServiceSecretRefs{
		"web": {{Source: "db_password"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, t.TempDir(), false, true)

	content := result[containerPath]
	expectedVolume := "Volume=" + secretFile + ":/run/secrets/db_password:ro,z"
	if !strings.Contains(content, expectedVolume) {
		t.Errorf("expected SELinux :z on volume, got:\n%s", content)
	}
}

func TestSecretHandler_EnvironmentSecretWithSELinux(t *testing.T) {
	dir := t.TempDir()
	secretsDir := t.TempDir()

	os.Setenv("TEST_TOKEN", "secret-token-value")
	defer os.Unsetenv("TEST_TOKEN")

	containerPath := writeTestFile(t, dir, "cq-myapp-api.container",
		"[Container]\nImage=docker.io/library/alpine:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"token": {Name: "token", Environment: "TEST_TOKEN"},
	}
	refs := preprocess.ServiceSecretRefs{
		"api": {{Source: "token"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, secretsDir, false, true)

	content := result[containerPath]
	expectedVolume := "Volume=" + filepath.Join(secretsDir, "token") + ":/run/secrets/token:ro,z"
	if !strings.Contains(content, expectedVolume) {
		t.Errorf("expected SELinux :z on volume, got:\n%s", content)
	}
}

func TestSecretHandler_SELinuxDoesNotAffectExternalSecrets(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n\n[Install]\nWantedBy=default.target\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	defs := map[string]preprocess.SecretDef{
		"apikey": {Name: "apikey", External: true},
	}
	refs := preprocess.ServiceSecretRefs{
		"web": {{Source: "apikey"}},
	}

	result := SecretHandler(files, "myapp", defs, refs, t.TempDir(), false, true)

	content := result[containerPath]
	if !strings.Contains(content, "Secret=apikey") {
		t.Error("expected Secret=apikey")
	}
	if strings.Contains(content, "Volume=") {
		t.Error("expected no Volume= for external secret")
	}
}

func TestInsertContainerDirective(t *testing.T) {
	lines := []string{
		"[Unit]",
		"After=network-online.target",
		"",
		"[Container]",
		"Image=nginx",
		"PublishPort=80:80",
		"",
		"[Install]",
		"WantedBy=default.target",
	}

	result := insertContainerDirective(lines, "Secret=apikey")

	content := strings.Join(result, "\n")
	expected := "[Unit]\nAfter=network-online.target\n\n[Container]\nImage=nginx\nPublishPort=80:80\nSecret=apikey\n\n[Install]\nWantedBy=default.target"
	if content != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", content, expected)
	}

	containerIdx := -1
	installIdx := -1
	secretIdx := -1
	for i, l := range result {
		trimmed := strings.TrimSpace(l)
		if trimmed == "[Container]" {
			containerIdx = i
		}
		if trimmed == "[Install]" {
			installIdx = i
		}
		if trimmed == "Secret=apikey" {
			secretIdx = i
		}
	}
	if containerIdx < 0 || installIdx < 0 || secretIdx < 0 {
		t.Fatal("expected all sections/directives present")
	}
	if secretIdx <= containerIdx || secretIdx >= installIdx {
		t.Error("Secret= must be between [Container] and [Install]")
	}
}

func TestInsertContainerDirective_NoBlankLineSeparation(t *testing.T) {
	lines := []string{
		"[Container]",
		"Image=nginx",
		"PublishPort=80:80",
		"",
		"[Install]",
		"WantedBy=default.target",
	}

	result := insertContainerDirective(lines, "Volume=/host/secrets/pass:/run/secrets/pass:ro")

	content := strings.Join(result, "\n")
	expected := "[Container]\nImage=nginx\nPublishPort=80:80\nVolume=/host/secrets/pass:/run/secrets/pass:ro\n\n[Install]\nWantedBy=default.target"
	if content != expected {
		t.Errorf("directive must be inside [Container] section without blank line separation:\ngot:  %q\nwant: %q", content, expected)
	}
}

func TestAddSELinuxToVolume(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Volume=/host/path:/run/secrets/pass:ro", "Volume=/host/path:/run/secrets/pass:ro,z"},
		{"Volume=src:dest", "Volume=src:dest:z"},
		{"Volume=src", "Volume=src:z"},
		{"Volume=src:dest:ro,z", "Volume=src:dest:ro,z"},
		{"Volume=src:dest:ro,Z", "Volume=src:dest:ro,Z"},
		{"Volume=src:dest:rw", "Volume=src:dest:rw,z"},
	}

	for _, tt := range tests {
		result := addSELinuxToVolume(tt.input)
		if result != tt.expected {
			t.Errorf("addSELinuxToVolume(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
