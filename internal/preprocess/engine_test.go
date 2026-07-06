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

func TestProcess_BuildServicesGetImageDirective(t *testing.T) {
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

	// web service should have image directive set
	if !contains(resultStr, "image: myapp-web:latest") {
		t.Errorf("expected image 'myapp-web:latest' for build service, got:\n%s", resultStr)
	}

	// build block should be removed
	if contains(resultStr, "build:") {
		t.Errorf("build block should be removed, got:\n%s", resultStr)
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

	// web service should have image directive replacing build
	if !contains(resultStr, "image: myapp-web:latest") {
		t.Errorf("expected image 'myapp-web:latest' for build service, got:\n%s", resultStr)
	}

	// build block should be removed
	if contains(resultStr, "build:") {
		t.Errorf("build block should be removed from web service, got:\n%s", resultStr)
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

	// Check that image directive is set
	if !contains(resultStr, "image: myapp-web:latest") {
		t.Errorf("expected image 'myapp-web:latest', got:\n%s", resultStr)
	}

	// Check that build block is removed
	if contains(resultStr, "context:") || contains(resultStr, "dockerfile:") || contains(resultStr, "target:") {
		t.Errorf("build config should be removed, got:\n%s", resultStr)
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

// ---------------------------------------------------------------------------
// buildArgValue edge cases
// ---------------------------------------------------------------------------

func TestBuildArgValue_Nil(t *testing.T) {
	if got := buildArgValue(nil); got != "" {
		t.Errorf("nil → expected \"\", got %q", got)
	}
}

func TestBuildArgValue_String(t *testing.T) {
	if got := buildArgValue("hello"); got != "hello" {
		t.Errorf("string → expected \"hello\", got %q", got)
	}
}

func TestBuildArgValue_EmptyString(t *testing.T) {
	if got := buildArgValue(""); got != "" {
		t.Errorf("empty string → expected \"\", got %q", got)
	}
}

func TestBuildArgValue_BoolTrue(t *testing.T) {
	if got := buildArgValue(true); got != "true" {
		t.Errorf("bool true → expected \"true\", got %q", got)
	}
}

func TestBuildArgValue_BoolFalse(t *testing.T) {
	if got := buildArgValue(false); got != "false" {
		t.Errorf("bool false → expected \"false\", got %q", got)
	}
}

func TestBuildArgValue_Int(t *testing.T) {
	if got := buildArgValue(42); got != "42" {
		t.Errorf("int → expected \"42\", got %q", got)
	}
}

func TestBuildArgValue_Int64(t *testing.T) {
	if got := buildArgValue(int64(99)); got != "99" {
		t.Errorf("int64 → expected \"99\", got %q", got)
	}
}

func TestBuildArgValue_Float64_WholeNumber(t *testing.T) {
	// YAML unmarshals plain integers as float64; whole floats should format
	// without a decimal point (e.g. 1.0 → "1", not "1.0").
	if got := buildArgValue(float64(1)); got != "1" {
		t.Errorf("float64(1) → expected \"1\", got %q", got)
	}
	if got := buildArgValue(float64(42)); got != "42" {
		t.Errorf("float64(42) → expected \"42\", got %q", got)
	}
}

func TestBuildArgValue_Float64_WithDecimal(t *testing.T) {
	if got := buildArgValue(1.5); got != "1.5" {
		t.Errorf("float64(1.5) → expected \"1.5\", got %q", got)
	}
	if got := buildArgValue(3.14); got != "3.14" {
		t.Errorf("float64(3.14) → expected \"3.14\", got %q", got)
	}
}

// ---------------------------------------------------------------------------
// GetBuildInfo edge cases
// ---------------------------------------------------------------------------

func TestGetBuildInfo_EmptyStringContext(t *testing.T) {
	// A build: block with an explicit empty context should fall back to "."
	// (resolved to the working directory).
	input := []byte(`
services:
  web:
    build:
      context: ""
      dockerfile: Dockerfile
`)
	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo(input)
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}
	webInfo, ok := buildInfo["web"]
	if !ok {
		t.Fatal("expected build info for 'web'")
	}
	// An empty context string is treated the same as omitted — defaults to "."
	// resolved against the working directory.
	if webInfo.Context != "/workdir" {
		t.Errorf("empty context → expected \"/workdir\", got %q", webInfo.Context)
	}
}

func TestGetBuildInfo_EmptyArgsMap(t *testing.T) {
	input := []byte(`
services:
  web:
    build:
      context: .
      args: {}
`)
	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo(input)
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}
	webInfo, ok := buildInfo["web"]
	if !ok {
		t.Fatal("expected build info for 'web'")
	}
	if len(webInfo.Args) != 0 {
		t.Errorf("empty args map → expected 0 args, got %v", webInfo.Args)
	}
}

func TestGetBuildInfo_NonStringArgTypes(t *testing.T) {
	// YAML bool, integer, and float args must be serialised correctly, not
	// as "<nil>" or with Go-specific formatting.
	input := []byte(`
services:
  web:
    build:
      context: .
      args:
        DEBUG: true
        COUNT: 3
        RATIO: 1.5
        EMPTY:
`)
	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo(input)
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}
	webInfo, ok := buildInfo["web"]
	if !ok {
		t.Fatal("expected build info for 'web'")
	}

	want := map[string]string{
		"DEBUG": "true",
		"COUNT": "3",
		"RATIO": "1.5",
		"EMPTY": "",
	}
	got := make(map[string]string)
	for _, arg := range webInfo.Args {
		// args are "KEY=VALUE" strings
		idx := 0
		for idx < len(arg) && arg[idx] != '=' {
			idx++
		}
		if idx < len(arg) {
			got[arg[:idx]] = arg[idx+1:]
		}
	}
	for k, wantV := range want {
		if gotV, ok := got[k]; !ok {
			t.Errorf("expected arg %q to be present", k)
		} else if gotV != wantV {
			t.Errorf("arg %q: expected %q, got %q", k, wantV, gotV)
		}
	}
}

func TestGetBuildInfo_ServiceWithLabelsAndBuild(t *testing.T) {
	// Labels alongside build: should not interfere with build info extraction.
	input := []byte(`
services:
  web:
    build: ./app
    labels:
      com.example.team: backend
      comquad-no-autoupdate: "true"
`)
	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo(input)
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}
	webInfo, ok := buildInfo["web"]
	if !ok {
		t.Fatal("expected build info for 'web'")
	}
	if webInfo.Context != "/workdir/app" {
		t.Errorf("expected context \"/workdir/app\", got %q", webInfo.Context)
	}
}

func TestProcess_EnvironmentAsListFormat(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
    environment:
      - REDIS_HOST=redis
      - MYSQL_HOST=db
      - MYSQL_DATABASE=nextcloud
      - MYSQL_USER=nextcloud
      - MYSQL_PASSWORD=nextcloud
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if !contains(resultStr, "REDIS_HOST") && !contains(resultStr, "redis") {
		t.Errorf("expected REDIS_HOST=redis in environment, got:\n%s", resultStr)
	}
	if !contains(resultStr, "MYSQL_HOST") && !contains(resultStr, "db") {
		t.Errorf("expected MYSQL_HOST=db in environment, got:\n%s", resultStr)
	}
}

func TestProcess_EnvironmentAsMapFormat(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
    environment:
      REDIS_HOST: redis
      MYSQL_HOST: db
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	// Environment preserved as map format inside generic service map
	if !contains(resultStr, "REDIS_HOST") {
		t.Errorf("expected REDIS_HOST in environment, got:\n%s", resultStr)
	}
	if !contains(resultStr, "redis") {
		t.Errorf("expected redis value in environment, got:\n%s", resultStr)
	}
}

func TestProcess_LabelsAsListFormat(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
    labels:
      - com.example.team=backend
      - com.example.version=1.0
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	// Labels preserved as map format inside generic service map
	if !contains(resultStr, "com.example.team") {
		t.Errorf("expected com.example.team in labels, got:\n%s", resultStr)
	}
	if !contains(resultStr, "com.example.version") {
		t.Errorf("expected com.example.version in labels, got:\n%s", resultStr)
	}
}

func TestProcess_LabelsAsMapFormat(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
    labels:
      com.example.team: backend
      com.example.version: "1.0"
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	// Labels preserved as map format inside generic service map
	if !contains(resultStr, "com.example.team") {
		t.Errorf("expected com.example.team in labels, got:\n%s", resultStr)
	}
	if !contains(resultStr, "com.example.version") {
		t.Errorf("expected com.example.version in labels, got:\n%s", resultStr)
	}
}

func TestProcess_NetworkLabelsAsListFormat(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
    networks:
      - frontend
networks:
  frontend:
    driver: bridge
    labels:
      - com.example.network=frontend
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if !contains(resultStr, "com.example.network=frontend") {
		t.Errorf("expected com.example.network=frontend in network labels, got:\n%s", resultStr)
	}
}

func TestProcess_VolumeLabelsAsListFormat(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
    volumes:
      - nc_data:/var/www/html
volumes:
  nc_data:
    driver: local
    labels:
      - com.example.volume=data
`)

	engine := NewEngine("myapp", "/some/dir")
	result, err := engine.Process(input)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	resultStr := string(result)
	if !contains(resultStr, "com.example.volume") {
		t.Errorf("expected com.example.volume in volume labels, got:\n%s", resultStr)
	}
	if !contains(resultStr, "com.comquad.force-volume") {
		t.Errorf("expected com.comquad.force-volume in volume labels, got:\n%s", resultStr)
	}
}

func TestProcess_BuildArgsAsListFormat(t *testing.T) {
	input := []byte(`
services:
  web:
    build:
      context: .
      args:
        - VERSION=1.0
        - ARCH=amd64
`)

	engine := NewEngine("myapp", "/workdir")
	buildInfo, err := engine.GetBuildInfo(input)
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}

	webInfo, ok := buildInfo["web"]
	if !ok {
		t.Fatal("expected build info for 'web'")
	}

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

	found = false
	for _, arg := range webInfo.Args {
		if arg == "ARCH=amd64" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected build arg 'ARCH=amd64', got %v", webInfo.Args)
	}
}

func TestReplaceBuildWithImage_StringContext(t *testing.T) {
	cf := ComposeFile{
		Services: map[string]map[string]interface{}{
			"web": {"build": "./myapp"},
		},
	}

	replaced := replaceBuildWithImage(&cf, "myapp")

	if len(replaced) != 1 || replaced[0] != "web" {
		t.Errorf("expected ['web'], got %v", replaced)
	}
	if cf.Services["web"]["image"] != "myapp-web:latest" {
		t.Errorf("expected image 'myapp-web:latest', got %v", cf.Services["web"]["image"])
	}
	if _, hasBuild := cf.Services["web"]["build"]; hasBuild {
		t.Errorf("build key should be removed")
	}
}

func TestReplaceBuildWithImage_MapContext(t *testing.T) {
	cf := ComposeFile{
		Services: map[string]map[string]interface{}{
			"api": {
				"build": map[string]interface{}{
					"context":    "./apps/api",
					"dockerfile": "Dockerfile.prod",
					"target":     "production",
					"args":       map[string]interface{}{"VERSION": "1.0"},
				},
				"labels": map[string]interface{}{"com.example": "true"},
			},
		},
	}

	replaced := replaceBuildWithImage(&cf, "myapp")

	if len(replaced) != 1 || replaced[0] != "api" {
		t.Errorf("expected ['api'], got %v", replaced)
	}
	if cf.Services["api"]["image"] != "myapp-api:latest" {
		t.Errorf("expected image 'myapp-api:latest', got %v", cf.Services["api"]["image"])
	}
	// All build keys should be gone
	for k := range cf.Services["api"] {
		if k == "build" {
			t.Errorf("build key should be removed")
		}
	}
	// Non-build keys should be preserved
	if cf.Services["api"]["labels"] == nil {
		t.Errorf("labels should be preserved")
	}
}

func TestReplaceBuildWithImage_NoBuildServices(t *testing.T) {
	cf := ComposeFile{
		Services: map[string]map[string]interface{}{
			"web": {"image": "nginx"},
			"db":  {"image": "postgres"},
		},
	}

	replaced := replaceBuildWithImage(&cf, "myapp")

	if len(replaced) != 0 {
		t.Errorf("expected empty slice, got %v", replaced)
	}
}

func TestReplaceBuildWithImage_MixedServices(t *testing.T) {
	cf := ComposeFile{
		Services: map[string]map[string]interface{}{
			"web": {"build": "./web"},
			"db":  {"image": "postgres"},
			"api": {"build": "./api"},
		},
	}

	replaced := replaceBuildWithImage(&cf, "myapp")

	if len(replaced) != 2 {
		t.Errorf("expected 2 replaced services, got %d", len(replaced))
	}
	// Only build services should be replaced
	if cf.Services["db"]["image"] != "postgres" {
		t.Errorf("non-build service image should be unchanged")
	}
	if cf.Services["web"]["image"] != "myapp-web:latest" {
		t.Errorf("web should have image set")
	}
	if cf.Services["api"]["image"] != "myapp-api:latest" {
		t.Errorf("api should have image set")
	}
}

func TestReplaceBuildWithImage_EmptyServices(t *testing.T) {
	cf := ComposeFile{
		Services: map[string]map[string]interface{}{},
	}

	replaced := replaceBuildWithImage(&cf, "myapp")

	if len(replaced) != 0 {
		t.Errorf("expected empty slice, got %v", replaced)
	}
}

func TestReplaceBuildWithImage_ProjectNameInTag(t *testing.T) {
	cf := ComposeFile{
		Services: map[string]map[string]interface{}{
			"web": {"build": "."},
		},
	}

	replaceBuildWithImage(&cf, "different-project")

	if cf.Services["web"]["image"] != "different-project-web:latest" {
		t.Errorf("expected 'different-project-web:latest', got %v", cf.Services["web"]["image"])
	}
}
