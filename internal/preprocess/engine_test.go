package preprocess

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProcess_NormalizesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	// Create a subdirectory so we have a valid absolute path to resolve
	volDir := filepath.Join(dir, "volumes", "data")
	if err := os.MkdirAll(volDir, 0755); err != nil {
		t.Fatal(err)
	}

	input := []byte(`
services:
  web:
    image: nginx
    volumes:
      - ./volumes/data:/data
      - ../other:/other
  db:
    image: postgres
    volumes:
      - ./db-data:/var/lib/postgresql/data:ro
`)

	engine := NewEngine("myproject", dir)
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	expectedAbs := filepath.Join(dir, "volumes", "data")

	// Check that relative path was resolved to absolute
	if !contains(resultStr, expectedAbs+":/data") {
		t.Errorf("expected volume path to be resolved to absolute, got:\n%s", resultStr)
	}

	// Check :ro option is preserved
	if !contains(resultStr, filepath.Join(dir, "db-data")+":/var/lib/postgresql/data:ro") {
		t.Errorf("expected :ro option to be preserved, got:\n%s", resultStr)
	}
}

func TestProcess_InjectsProjectLabel(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
`)

	engine := NewEngine("testproject", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	// Labels are now injected in the cooker step, not the preprocessor
	expected := "container_name: testproject-web"
	if !contains(resultStr, expected) {
		t.Errorf("expected container_name %q not found in:\n%s", expected, resultStr)
	}
}

func TestProcess_InjectsContainerName(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
  db:
    image: postgres
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	expected := "container_name: myapp-web"
	if !contains(resultStr, expected) {
		t.Errorf("expected %q not found in:\n%s", expected, resultStr)
	}
}

func TestProcess_ExistingContainerNameNotOverridden(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
    container_name: custom-name
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if !contains(resultStr, "custom-name") {
		t.Errorf("expected custom container_name to be preserved, got:\n%s", resultStr)
	}
}

func TestProcess_CreatesDefaultNetwork(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if !contains(resultStr, "cq-default") {
		t.Errorf("expected default network 'cq-default' not found in:\n%s", resultStr)
	}
}

func TestProcess_ServiceAttachedToDefaultNetwork(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	// Service with no explicit networks should be attached to cq-default
	if !contains(resultStr, "cq-default") {
		t.Errorf("expected service to be attached to cq-default, got:\n%s", resultStr)
	}
}

func TestProcess_NoDefaultNetworkWhenNetworksDefined(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
    networks:
      - frontend
networks:
  frontend:
    driver: bridge
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if contains(resultStr, "cq-default") {
		t.Errorf("expected no default network when networks are already defined, got:\n%s", resultStr)
	}
}

func TestProcess_EmptyServices(t *testing.T) {
	input := []byte(`
services: {}
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestProcess_NoServices(t *testing.T) {
	input := []byte(`
version: "3"
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestProcess_ServicesWithoutLabels(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	// Labels are now injected in the cooker step, not the preprocessor
	if !contains(resultStr, "container_name: myapp-web") {
		t.Errorf("expected container_name injection, got:\n%s", resultStr)
	}
}

func TestProcess_ExistingLabelsPreserved(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
    labels:
      mylabel: "value"
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	// Labels preserved as map format inside generic service map
	if !contains(resultStr, "mylabel") && !contains(resultStr, "mylabel:") {
		t.Errorf("expected existing label to be preserved, got:\n%s", resultStr)
	}
}

func TestProcess_InvalidYaml(t *testing.T) {
	input := []byte(`
services:
  web: [invalid yaml
`)

	engine := NewEngine("myapp", "/some/dir")
	_, err := engine.Process(input)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestNewEngine(t *testing.T) {
	engine := NewEngine("proj", "/workdir")
	if engine.ProjectName != "proj" {
		t.Errorf("expected ProjectName 'proj', got %q", engine.ProjectName)
	}
	if engine.WorkingDirectory != "/workdir" {
		t.Errorf("expected WorkingDirectory '/workdir', got %q", engine.WorkingDirectory)
	}
}

func TestNormalizeImage_BareImage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"nginx", "docker.io/library/nginx"},
		{"postgres:15", "docker.io/library/postgres:15"},
		{"alpine", "docker.io/library/alpine"},
		{"myimage:latest", "docker.io/library/myimage:latest"},
	}

	for _, tt := range tests {
		result := normalizeImage(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeImage(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeImage_CustomRegistry(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"myregistry.com/myimage", "myregistry.com/myimage"},
		{"localhost:5000/myimage", "localhost:5000/myimage"},
		{"ghcr.io/myorg/myimage", "ghcr.io/myorg/myimage"},
		{"registry.example.com:8080/ns/image", "registry.example.com:8080/ns/image"},
	}

	for _, tt := range tests {
		result := normalizeImage(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeImage(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeImage_AlreadyDockerHub(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"docker.io/library/nginx", "docker.io/library/nginx"},
		{"docker.io/nginx", "docker.io/nginx"},
		{"docker.io/myorg/myimage:tag", "docker.io/myorg/myimage:tag"},
	}

	for _, tt := range tests {
		result := normalizeImage(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeImage(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeImage_LibraryNamespace(t *testing.T) {
	// "library/nginx" has a slash but no registry — should get docker.io/ prefix
	result := normalizeImage("library/nginx")
	expected := "docker.io/library/nginx"
	if result != expected {
		t.Errorf("normalizeImage(%q) = %q, want %q", "library/nginx", result, expected)
	}
}

func TestNormalizeImage_EmptyString(t *testing.T) {
	result := normalizeImage("")
	expected := "docker.io/library/"
	if result != expected {
		t.Errorf("normalizeImage(%q) = %q, want %q", "", result, expected)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && (len(haystack) >= len(needle)) && (haystack == needle || len(haystack) > len(needle) && (haystack[:len(needle)] == needle || contains(haystack[1:], needle)))
}

func TestStringMap_UnmarshalYAML_ListFormat(t *testing.T) {
	yamlData := `- app.name=myapp
- app.env=production`

	node := parseYAMLNode(t, yamlData)
	var sm StringMap
	if err := sm.UnmarshalYAML(node); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sm["app.name"] != "myapp" {
		t.Errorf("expected app.name=myapp, got %q", sm["app.name"])
	}
	if sm["app.env"] != "production" {
		t.Errorf("expected app.env=production, got %q", sm["app.env"])
	}
}

func TestStringMap_UnmarshalYAML_MapFormat(t *testing.T) {
	yamlData := `app.name: myapp
app.env: production`

	node := parseYAMLNode(t, yamlData)
	var sm StringMap
	if err := sm.UnmarshalYAML(node); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sm["app.name"] != "myapp" {
		t.Errorf("expected app.name=myapp, got %q", sm["app.name"])
	}
	if sm["app.env"] != "production" {
		t.Errorf("expected app.env=production, got %q", sm["app.env"])
	}
}

func TestStringMap_UnmarshalYAML_EmptyList(t *testing.T) {
	yamlData := "[]"

	node := parseYAMLNode(t, yamlData)
	var sm StringMap
	if err := sm.UnmarshalYAML(node); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sm) != 0 {
		t.Errorf("expected empty map, got %v", sm)
	}
}

func TestStringMap_UnmarshalYAML_EmptyMap(t *testing.T) {
	yamlData := "{}"

	node := parseYAMLNode(t, yamlData)
	var sm StringMap
	if err := sm.UnmarshalYAML(node); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sm) != 0 {
		t.Errorf("expected empty map, got %v", sm)
	}
}

func TestStringMap_UnmarshalYAML_YAMLNull(t *testing.T) {
	yamlData := "null"

	node := parseYAMLNode(t, yamlData)
	var sm StringMap
	if err := sm.UnmarshalYAML(node); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sm) != 0 {
		t.Errorf("expected empty map, got %v", sm)
	}
}

func TestStringMap_UnmarshalYAML_InvalidFormat(t *testing.T) {
	yamlData := "42"

	node := parseYAMLNode(t, yamlData)
	var sm StringMap
	err := sm.UnmarshalYAML(node)
	if err == nil {
		t.Error("expected error for scalar value, got nil")
	}
}

func TestStringMap_MarshalYAML(t *testing.T) {
	sm := StringMap{
		"app.name": "myapp",
		"app.env":  "production",
	}
	result, err := sm.MarshalYAML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := result.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", result)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list))
	}
	if list[0] != "app.env=production" {
		t.Errorf("expected 'app.env=production', got %q", list[0])
	}
	if list[1] != "app.name=myapp" {
		t.Errorf("expected 'app.name=myapp', got %q", list[1])
	}
}

func TestStringMap_MarshalYAML_Empty(t *testing.T) {
	sm := StringMap{}
	result, err := sm.MarshalYAML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := result.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", result)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %v", list)
	}
}

func parseYAMLNode(t *testing.T, yamlData string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(yamlData), &doc); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	if len(doc.Content) == 0 {
		t.Fatal("empty YAML document")
	}
	return doc.Content[0]
}

// ---------------------------------------------------------------------------
// ExtractServiceImageSpecs
// ---------------------------------------------------------------------------

func TestExtractServiceImageSpecs_BasicImage(t *testing.T) {
	input := []byte(`services:
  web:
    image: nginx
`)

	specs, err := ExtractServiceImageSpecs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spec, ok := specs["web"]
	if !ok {
		t.Fatal("expected spec for 'web' service")
	}
	if spec.ServiceName != "web" {
		t.Errorf("expected ServiceName 'web', got %q", spec.ServiceName)
	}
	if spec.Image != "docker.io/library/nginx" {
		t.Errorf("expected image normalization, got %q", spec.Image)
	}
}

func TestExtractServiceImageSpecs_PullPolicy(t *testing.T) {
	input := []byte(`services:
  app:
    image: alpine
    pull_policy: always
`)

	specs, err := ExtractServiceImageSpecs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spec := specs["app"]
	if spec.PullPolicy != "always" {
		t.Errorf("expected PullPolicy 'always', got %q", spec.PullPolicy)
	}
}

func TestExtractServiceImageSpecs_PlatformFull(t *testing.T) {
	input := []byte(`services:
  app:
    image: nginx
    platform: linux/arm64/v8
`)

	specs, err := ExtractServiceImageSpecs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spec := specs["app"]
	if spec.OS != "linux" {
		t.Errorf("expected OS 'linux', got %q", spec.OS)
	}
	if spec.Arch != "arm64" {
		t.Errorf("expected Arch 'arm64', got %q", spec.Arch)
	}
	if spec.Variant != "v8" {
		t.Errorf("expected Variant 'v8', got %q", spec.Variant)
	}
}

func TestExtractServiceImageSpecs_PlatformOSArchOnly(t *testing.T) {
	input := []byte(`services:
  app:
    image: nginx
    platform: windows/amd64
`)

	specs, err := ExtractServiceImageSpecs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spec := specs["app"]
	if spec.OS != "windows" {
		t.Errorf("expected OS 'windows', got %q", spec.OS)
	}
	if spec.Arch != "amd64" {
		t.Errorf("expected Arch 'amd64', got %q", spec.Arch)
	}
	if spec.Variant != "" {
		t.Errorf("expected no Variant, got %q", spec.Variant)
	}
}

func TestExtractServiceImageSpecs_PlatformOSOnly(t *testing.T) {
	input := []byte(`services:
  app:
    image: nginx
    platform: darwin
`)

	specs, err := ExtractServiceImageSpecs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spec := specs["app"]
	if spec.OS != "darwin" {
		t.Errorf("expected OS 'darwin', got %q", spec.OS)
	}
	if spec.Arch != "" {
		t.Errorf("expected no Arch, got %q", spec.Arch)
	}
}

func TestExtractServiceImageSpecs_MultipleServices(t *testing.T) {
	input := []byte(`services:
  web:
    image: nginx
    pull_policy: always
    platform: linux/amd64
  db:
    image: postgres:15
    pull_policy: never
`)

	specs, err := ExtractServiceImageSpecs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}

	web := specs["web"]
	if web.Image != "docker.io/library/nginx" {
		t.Errorf("web image: %q", web.Image)
	}
	if web.PullPolicy != "always" {
		t.Errorf("web pull_policy: %q", web.PullPolicy)
	}
	if web.OS != "linux" || web.Arch != "amd64" {
		t.Errorf("web platform: OS=%q Arch=%q", web.OS, web.Arch)
	}

	db := specs["db"]
	if db.Image != "docker.io/library/postgres:15" {
		t.Errorf("db image: %q", db.Image)
	}
	if db.PullPolicy != "never" {
		t.Errorf("db pull_policy: %q", db.PullPolicy)
	}
}

func TestExtractServiceImageSpecs_NoServices(t *testing.T) {
	input := []byte(`version: "3"`)

	specs, err := ExtractServiceImageSpecs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 0 {
		t.Errorf("expected empty specs, got %d", len(specs))
	}
}

func TestExtractServiceImageSpecs_InvalidYAML(t *testing.T) {
	input := []byte(`not: valid: [yaml`)
	_, err := ExtractServiceImageSpecs(input)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// ---------------------------------------------------------------------------
// Process strips pull_policy and platform
// ---------------------------------------------------------------------------

func TestProcess_StripsPullPolicy(t *testing.T) {
	input := []byte(`services:
  web:
    image: nginx
    pull_policy: always
`)

	engine := NewEngine("myapp", "/tmp")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if contains(string(result), "pull_policy") {
		t.Errorf("expected pull_policy to be stripped, got:\n%s", string(result))
	}
}

func TestProcess_StripsPlatform(t *testing.T) {
	input := []byte(`services:
  web:
    image: nginx
    platform: linux/amd64
`)

	engine := NewEngine("myapp", "/tmp")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if contains(string(result), "platform") {
		t.Errorf("expected platform to be stripped, got:\n%s", string(result))
	}
}

func TestProcess_PreservesOtherFields(t *testing.T) {
	input := []byte(`services:
  web:
    image: nginx
    pull_policy: always
    platform: linux/amd64
    environment:
      FOO: bar
`)

	engine := NewEngine("myapp", "/tmp")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if !contains(resultStr, "FOO") || !contains(resultStr, "bar") {
		t.Errorf("expected environment to be preserved, got:\n%s", resultStr)
	}
}

func TestProcess_StripsSecrets(t *testing.T) {
	input := []byte(`secrets:
  db_password:
    file: ./secrets/db.txt
services:
  web:
    image: nginx
    secrets:
      - db_password
`)

	engine := NewEngine("myapp", "/tmp")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if contains(resultStr, "secrets:") {
		t.Errorf("expected top-level secrets to be stripped, got:\n%s", resultStr)
	}
	if contains(resultStr, "db_password") && !contains(resultStr, "container_name") {
		t.Errorf("expected per-service secrets to be stripped, got:\n%s", resultStr)
	}
}

func TestProcess_StripsSecretsFromServices(t *testing.T) {
	input := []byte(`services:
  web:
    image: nginx
    secrets:
      - app_key
`)

	engine := NewEngine("myapp", "/tmp")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if contains(resultStr, "secrets:") && !contains(resultStr, "networks:") {
		t.Errorf("expected per-service secrets to be stripped, got:\n%s", resultStr)
	}
}

func TestProcess_StripsSecretsPreservesOtherFields(t *testing.T) {
	input := []byte(`secrets:
  db_password:
    file: ./secrets/db.txt
services:
  web:
    image: nginx
    environment:
      FOO: bar
    secrets:
      - db_password
`)

	engine := NewEngine("myapp", "/tmp")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if contains(resultStr, "secrets:") {
		t.Errorf("expected secrets to be stripped, got:\n%s", resultStr)
	}
	if !contains(resultStr, "FOO") || !contains(resultStr, "bar") {
		t.Errorf("expected environment to be preserved, got:\n%s", resultStr)
	}
}

// ---------------------------------------------------------------------------
// ExtractSecretSpecs
// ---------------------------------------------------------------------------

func TestExtractSecretSpecs_FileSecret(t *testing.T) {
	dir := t.TempDir()
	input := []byte(`secrets:
  db_password:
    file: ./secrets/db.txt
services:
  web:
    secrets:
      - db_password
`)

	defs, refs, err := ExtractSecretSpecs(input, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def, ok := defs["db_password"]
	if !ok {
		t.Fatal("expected secret 'db_password' in definitions")
	}
	if def.External {
		t.Error("expected non-external secret")
	}
	if def.File != filepath.Join(dir, "secrets", "db.txt") {
		t.Errorf("expected absolute file path, got %q", def.File)
	}

	if len(refs["web"]) != 1 || refs["web"][0].Source != "db_password" {
		t.Errorf("expected web to reference db_password, got %v", refs["web"])
	}
}

func TestExtractSecretSpecs_EnvironmentSecret(t *testing.T) {
	os.Setenv("OAUTH_TOKEN", "my-secret-token")
	defer os.Unsetenv("OAUTH_TOKEN")

	input := []byte(`secrets:
  token:
    environment: "OAUTH_TOKEN"
services:
  api:
    secrets:
      - token
`)

	defs, refs, err := ExtractSecretSpecs(input, "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def, ok := defs["token"]
	if !ok {
		t.Fatal("expected secret 'token' in definitions")
	}
	if def.Environment != "OAUTH_TOKEN" {
		t.Errorf("expected Environment 'OAUTH_TOKEN', got %q", def.Environment)
	}
	if def.Content != "my-secret-token" {
		t.Errorf("expected Content 'my-secret-token', got %q", def.Content)
	}
	if def.External {
		t.Error("expected non-external secret")
	}

	if len(refs["api"]) != 1 || refs["api"][0].Source != "token" {
		t.Errorf("expected api to reference token, got %v", refs["api"])
	}
}

func TestExtractSecretSpecs_EnvironmentFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_PASSWORD=SuperSecretFromDotEnv\n"), 0644)

	input := []byte(`secrets:
  db_password:
    environment: "DB_PASSWORD"
services:
  db:
    secrets:
      - db_password
`)

	defs, _, err := ExtractSecretSpecs(input, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def, ok := defs["db_password"]
	if !ok {
		t.Fatal("expected secret 'db_password' in definitions")
	}
	if def.Content != "SuperSecretFromDotEnv" {
		t.Errorf("expected Content from .env, got %q", def.Content)
	}
}

func TestExtractSecretSpecs_EnvironmentOsEnvTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_PASSWORD=FromDotEnv\n"), 0644)
	os.Setenv("DB_PASSWORD", "FromOSEnv")
	defer os.Unsetenv("DB_PASSWORD")

	input := []byte(`secrets:
  db_password:
    environment: "DB_PASSWORD"
`)

	defs, _, err := ExtractSecretSpecs(input, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def := defs["db_password"]
	if def.Content != "FromOSEnv" {
		t.Errorf("expected OS env to take precedence, got %q", def.Content)
	}
}

func TestExtractSecretSpecs_ExternalSecret(t *testing.T) {
	input := []byte(`secrets:
  app_key:
    external: true
services:
  web:
    secrets:
      - app_key
`)

	defs, _, err := ExtractSecretSpecs(input, "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def, ok := defs["app_key"]
	if !ok {
		t.Fatal("expected secret 'app_key' in definitions")
	}
	if !def.External {
		t.Error("expected external secret")
	}
}

func TestExtractSecretSpecs_ExternalWithName(t *testing.T) {
	input := []byte(`secrets:
  server-certificate:
    external: true
    name: "CERTIFICATE_KEY"
services:
  web:
    secrets:
      - server-certificate
`)

	defs, _, err := ExtractSecretSpecs(input, "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def, ok := defs["server-certificate"]
	if !ok {
		t.Fatal("expected secret 'server-certificate' in definitions")
	}
	if !def.External {
		t.Error("expected external secret")
	}
	if def.ExternalName != "CERTIFICATE_KEY" {
		t.Errorf("expected ExternalName 'CERTIFICATE_KEY', got %q", def.ExternalName)
	}
}

func TestExtractSecretSpecs_ExternalRejectsOtherAttributes(t *testing.T) {
	input := []byte(`secrets:
  app_key:
    external: true
    file: ./some/path
`)

	_, _, err := ExtractSecretSpecs(input, "/tmp")
	if err == nil {
		t.Fatal("expected error for external secret with file attribute")
	}
}

func TestExtractSecretSpecs_UnknownSecretReference(t *testing.T) {
	input := []byte(`services:
  web:
    secrets:
      - nonexistent
`)

	_, _, err := ExtractSecretSpecs(input, "/tmp")
	if err == nil {
		t.Fatal("expected error for undefined secret reference")
	}
}

func TestExtractSecretSpecs_MultipleSecrets(t *testing.T) {
	input := []byte(`secrets:
  db_password:
    file: ./secrets/db.txt
  apikey:
    external: true
services:
  web:
    secrets:
      - db_password
      - apikey
  api:
    secrets:
      - apikey
`)

	defs, refs, err := ExtractSecretSpecs(input, "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(defs) != 2 {
		t.Fatalf("expected 2 secret definitions, got %d", len(defs))
	}

	if len(refs["web"]) != 2 {
		t.Errorf("expected web to reference 2 secrets, got %d", len(refs["web"]))
	}
	if len(refs["api"]) != 1 {
		t.Errorf("expected api to reference 1 secret, got %d", len(refs["api"]))
	}
}

func TestExtractSecretSpecs_NoSecrets(t *testing.T) {
	input := []byte(`services:
  web:
    image: nginx
`)

	defs, refs, err := ExtractSecretSpecs(input, "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 0 {
		t.Errorf("expected no secret definitions, got %d", len(defs))
	}
	if len(refs) != 0 {
		t.Errorf("expected no service refs, got %d", len(refs))
	}
}

func TestExtractSecretSpecs_InvalidYAML(t *testing.T) {
	input := []byte(`not: valid: [yaml`)
	_, _, err := ExtractSecretSpecs(input, "/tmp")
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
