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
