package preprocess

import (
	"os"
	"path/filepath"
	"testing"
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
	if !contains(resultStr, "mylabel: value") {
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

func TestProcess_SkipsImageNormalizationForBuildServices(t *testing.T) {
	input := []byte(`
services:
  web:
    build: .
  db:
    image: postgres
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)

	// web service should NOT have docker.io/library/ prefix
	if contains(resultStr, "docker.io/library/web") {
		t.Errorf("build service 'web' should not have docker.io/library/ prefix")
	}

	// db service should have docker.io/library/ prefix
	if !contains(resultStr, "docker.io/library/postgres") {
		t.Errorf("image service 'db' should have docker.io/library/ prefix")
	}
}

func TestGetBuildInfo_StringContext(t *testing.T) {
	input := []byte(`
services:
  web:
    build: ./myapp
  api:
    build: .
`)

	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo([]byte(input))
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}

	webInfo, ok := buildInfo["web"]
	if !ok {
		t.Fatal("expected build info for 'web' service")
	}

	if webInfo.Context != "/workdir/myapp" {
		t.Errorf("expected web context '/workdir/myapp', got %q", webInfo.Context)
	}

	if webInfo.Dockerfile != "Dockerfile" {
		t.Errorf("expected web dockerfile 'Dockerfile', got %q", webInfo.Dockerfile)
	}

	apiInfo, ok := buildInfo["api"]
	if !ok {
		t.Fatal("expected build info for 'api' service")
	}

	if apiInfo.Context != "/workdir" {
		t.Errorf("expected api context '/workdir', got %q", apiInfo.Context)
	}
}

func TestGetBuildInfo_MapContext(t *testing.T) {
	input := []byte(`
services:
  web:
    build:
      context: ./apps/web
      dockerfile: Dockerfile.prod
      target: production
      args:
        VERSION: "1.0"
        ARCH: "amd64"
`)

	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo([]byte(input))
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}

	webInfo, ok := buildInfo["web"]
	if !ok {
		t.Fatal("expected build info for 'web' service")
	}

	if webInfo.Context != "/workdir/apps/web" {
		t.Errorf("expected web context '/workdir/apps/web', got %q", webInfo.Context)
	}

	if webInfo.Dockerfile != "Dockerfile.prod" {
		t.Errorf("expected web dockerfile 'Dockerfile.prod', got %q", webInfo.Dockerfile)
	}

	if webInfo.Target != "production" {
		t.Errorf("expected web target 'production', got %q", webInfo.Target)
	}

	// Check build args
	found := false
	for _, arg := range webInfo.Args {
		if arg == "VERSION=1.0" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected build arg 'VERSION=1.0', got %v", webInfo.Args)
	}
}

func TestGetBuildInfo_NoBuildServices(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
  db:
    image: postgres
`)

	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo([]byte(input))
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}

	if len(buildInfo) != 0 {
		t.Errorf("expected no build info, got %d entries", len(buildInfo))
	}
}

func TestProcess_BuildWithStringContext(t *testing.T) {
	input := []byte(`
services:
  web:
    build: ./myapp
  db:
    image: postgres
`)

	engine := NewEngine("myapp", "/workdir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)

	// web service should have build context preserved (converted to map form)
	if !contains(resultStr, "context: ./myapp") {
		t.Errorf("expected build context './myapp' to be preserved, got:\n%s", resultStr)
	}

	// web service should NOT have docker.io/library/ prefix
	if contains(resultStr, "docker.io/library/web") {
		t.Errorf("build service 'web' should not have docker.io/library/ prefix")
	}

	// db service should have docker.io/library/ prefix
	if !contains(resultStr, "docker.io/library/postgres") {
		t.Errorf("image service 'db' should have docker.io/library/ prefix")
	}
}

func TestProcess_BuildWithMapContext(t *testing.T) {
	input := []byte(`
services:
  web:
    build:
      context: ./apps/web
      dockerfile: Dockerfile.prod
      target: production
`)

	engine := NewEngine("myapp", "/workdir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)

	// Check that build config is preserved
	if !contains(resultStr, "context: ./apps/web") {
		t.Errorf("expected context to be preserved")
	}
	if !contains(resultStr, "dockerfile: Dockerfile.prod") {
		t.Errorf("expected dockerfile to be preserved")
	}
	if !contains(resultStr, "target: production") {
		t.Errorf("expected target to be preserved")
	}
}

func TestGetBuildInfo_BuildWithStringContext(t *testing.T) {
	input := []byte(`
services:
  web:
    build: ./myapp
`)

	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo([]byte(input))
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}

	webInfo, ok := buildInfo["web"]
	if !ok {
		t.Fatal("expected build info for 'web' service")
	}

	if webInfo.Context != "/workdir/myapp" {
		t.Errorf("expected context '/workdir/myapp', got %q", webInfo.Context)
	}

	if webInfo.Dockerfile != "Dockerfile" {
		t.Errorf("expected dockerfile 'Dockerfile', got %q", webInfo.Dockerfile)
	}
}

func TestGetBuildInfo_BuildWithMapContext(t *testing.T) {
	input := []byte(`
services:
  web:
    build:
      context: ./apps/web
      dockerfile: Dockerfile.prod
      target: prod
      args:
        VERSION: "1.0"
`)

	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo([]byte(input))
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}

	webInfo, ok := buildInfo["web"]
	if !ok {
		t.Fatal("expected build info for 'web' service")
	}

	if webInfo.Context != "/workdir/apps/web" {
		t.Errorf("expected context '/workdir/apps/web', got %q", webInfo.Context)
	}

	if webInfo.Dockerfile != "Dockerfile.prod" {
		t.Errorf("expected dockerfile 'Dockerfile.prod', got %q", webInfo.Dockerfile)
	}

	if webInfo.Target != "prod" {
		t.Errorf("expected target 'prod', got %q", webInfo.Target)
	}
}

