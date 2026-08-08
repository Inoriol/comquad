package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Inoriol/comquad/internal/preprocess"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeTestFile: %v", err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile: %v", err)
	}
	return string(data)
}

// ---------------------------------------------------------------------------
// ImageQuadletHandler
// ---------------------------------------------------------------------------

func TestImageQuadletHandler_CreatesImageFile(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\nPublishPort=80:80\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web", Image: "docker.io/library/nginx:latest"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	imageContent, ok := result[imageFile]
	if !ok {
		t.Fatal("expected .image file in result map")
	}
	if !strings.Contains(imageContent, "[Image]") {
		t.Error("expected [Image] section in .image file")
	}
	if !strings.Contains(imageContent, "Image=docker.io/library/nginx:latest") {
		t.Error("expected Image= in .image file")
	}
	if _, err := os.Stat(imageFile); os.IsNotExist(err) {
		t.Error("expected .image file on disk")
	}
}

func TestImageQuadletHandler_UpdatesContainerReference(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\nPublishPort=80:80\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	containerContent, ok := result[containerPath]
	if !ok {
		t.Fatal("expected container file in result map")
	}
	if !strings.Contains(containerContent, "Image=cq-myapp-web.image") {
		t.Errorf("expected Image=cq-myapp-web.image in container file, got:\n%s", containerContent)
	}
	if strings.Contains(containerContent, "Image=docker.io/library/nginx") {
		t.Error("container file should not contain the raw image reference")
	}
}

func TestImageQuadletHandler_AddsPolicyWhenPresent(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web", PullPolicy: "always"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	imageContent := result[imageFile]
	if !strings.Contains(imageContent, "Policy=always") {
		t.Errorf("expected Policy=always in .image file, got:\n%s", imageContent)
	}
}

func TestImageQuadletHandler_MapsIfNotPresentToMissing(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web", PullPolicy: "if_not_present"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	imageContent := result[imageFile]
	if !strings.Contains(imageContent, "Policy=missing") {
		t.Errorf("expected Policy=missing for if_not_present, got:\n%s", imageContent)
	}
}

func TestImageQuadletHandler_OmitUnsupportedPullPolicy(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web", PullPolicy: "build"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	imageContent := result[imageFile]
	if strings.Contains(imageContent, "Policy=") {
		t.Errorf("expected no Policy= for 'build' pull_policy, got:\n%s", imageContent)
	}
}

func TestImageQuadletHandler_AddsPlatformFields(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web", OS: "linux", Arch: "amd64"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	imageContent := result[imageFile]
	if !strings.Contains(imageContent, "OS=linux") {
		t.Errorf("expected OS=linux in .image file, got:\n%s", imageContent)
	}
	if !strings.Contains(imageContent, "Arch=amd64") {
		t.Errorf("expected Arch=amd64 in .image file, got:\n%s", imageContent)
	}
}

func TestImageQuadletHandler_AddsVariant(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web", OS: "linux", Arch: "arm64", Variant: "v8"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	imageContent := result[imageFile]
	if !strings.Contains(imageContent, "Variant=v8") {
		t.Errorf("expected Variant=v8 in .image file, got:\n%s", imageContent)
	}
}

func TestImageQuadletHandler_AddsDefaults(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	imageContent := result[imageFile]
	if !strings.Contains(imageContent, "Retry=3") {
		t.Error("expected Retry=3 in .image file")
	}
	if !strings.Contains(imageContent, "RetryDelay=5s") {
		t.Error("expected RetryDelay=5s in .image file")
	}
	if strings.Contains(imageContent, "AutoUpdate=") {
		t.Error("AutoUpdate= is not valid in [Image] section")
	}
}

func TestImageQuadletHandler_NoContainerLabelsInImage(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	imageContent := result[imageFile]
	if strings.Contains(imageContent, "Label=com.comquad.project=") {
		t.Error("Label= is not valid in [Image] section")
	}
	if strings.Contains(imageContent, "Label=com.comquad.managed=") {
		t.Error("Label= is not valid in [Image] section")
	}
}

func TestImageQuadletHandler_AddsInstallSection(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	imageContent := result[imageFile]
	if !strings.Contains(imageContent, "[Install]") {
		t.Error("expected [Install] section in .image file")
	}
	if !strings.Contains(imageContent, "WantedBy=default.target") {
		t.Error("expected WantedBy=default.target in .image file")
	}
}

func TestImageQuadletHandler_SkipsNonContainerFiles(t *testing.T) {
	dir := t.TempDir()

	netPath := writeTestFile(t, dir, "cq-myapp-default.network", "[Network]\n")

	files := map[string]string{
		netPath: readFile(t, netPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	if len(result) != 1 {
		t.Errorf("expected 1 file in result, got %d", len(result))
	}
	if _, ok := result[strings.Replace(netPath, ".network", ".image", 1)]; ok {
		t.Error("expected no .image file for network")
	}
}

func TestImageQuadletHandler_MultipleServices(t *testing.T) {
	dir := t.TempDir()

	webPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")
	dbPath := writeTestFile(t, dir, "cq-myapp-db.container",
		"[Container]\nImage=docker.io/library/postgres:15\n")

	files := map[string]string{
		webPath: readFile(t, webPath),
		dbPath:  readFile(t, dbPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web"},
		"db":  {ServiceName: "db"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	if len(result) != 4 {
		t.Errorf("expected 4 files (2 containers + 2 images), got %d", len(result))
	}

	webImage := filepath.Join(dir, "cq-myapp-web.image")
	dbImage := filepath.Join(dir, "cq-myapp-db.image")

	if _, ok := result[webImage]; !ok {
		t.Error("expected web .image file")
	}
	if _, ok := result[dbImage]; !ok {
		t.Error("expected db .image file")
	}

	webContainer := result[webPath]
	if !strings.Contains(webContainer, "Image=cq-myapp-web.image") {
		t.Error("web container should reference web .image")
	}
	dbContainer := result[dbPath]
	if !strings.Contains(dbContainer, "Image=cq-myapp-db.image") {
		t.Error("db container should reference db .image")
	}
}

func TestImageQuadletHandler_NoImageInContainer(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nPublishPort=80:80\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web", Image: "docker.io/library/nginx:latest"},
	}

	result := ImageQuadletHandler(files, "myapp", services)

	imageFile := filepath.Join(dir, "cq-myapp-web.image")
	if _, ok := result[imageFile]; ok {
		t.Error("expected no .image file when container has no Image=")
	}
}

func TestImageQuadletHandler_UnknownServiceSkips(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-unknown.container",
		"[Container]\nImage=docker.io/library/nginx:latest\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{}

	result := ImageQuadletHandler(files, "myapp", services)

	if len(result) != 1 {
		t.Errorf("expected 1 file (unchanged container), got %d", len(result))
	}
}

func TestImageQuadletHandler_ContainerOnDiskUpdated(t *testing.T) {
	dir := t.TempDir()

	containerPath := writeTestFile(t, dir, "cq-myapp-web.container",
		"[Container]\nImage=docker.io/library/nginx:latest\nPublishPort=80:80\n")

	files := map[string]string{
		containerPath: readFile(t, containerPath),
	}

	services := map[string]preprocess.ServiceImageSpec{
		"web": {ServiceName: "web"},
	}

	ImageQuadletHandler(files, "myapp", services)

	diskContent := readFile(t, containerPath)
	if !strings.Contains(diskContent, "Image=cq-myapp-web.image") {
		t.Errorf("expected container on disk to reference .image, got:\n%s", diskContent)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		fileName    string
		projectPref string
		want        string
	}{
		{"cq-myapp-web.container", "cq-myapp-", "web"},
		{"cq-myproj-db.container", "cq-myproj-", "db"},
		{"cq-foo-bar-baz.container", "cq-foo-", "bar-baz"},
	}

	for _, tt := range tests {
		got := extractServiceName(tt.fileName, tt.projectPref)
		if got != tt.want {
			t.Errorf("extractServiceName(%q, %q) = %q, want %q",
				tt.fileName, tt.projectPref, got, tt.want)
		}
	}
}

func TestMakeImageFileName(t *testing.T) {
	got := makeImageFileName("/tmp/systemd/cq-myapp-web.container")
	want := "/tmp/systemd/cq-myapp-web.image"
	if got != want {
		t.Errorf("makeImageFileName = %q, want %q", got, want)
	}
}

func TestImageBaseName(t *testing.T) {
	got := imageBaseName("/tmp/systemd/cq-myapp-web.container")
	want := "cq-myapp-web.image"
	if got != want {
		t.Errorf("imageBaseName = %q, want %q", got, want)
	}
}

func TestComposePolicyToQuadlet(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"always", "always"},
		{"ALWAYS", "always"},
		{"never", "never"},
		{"missing", "missing"},
		{"if_not_present", "missing"},
		{"If_Not_Present", "missing"},
		{"build", ""},
		{"daily", ""},
		{"weekly", ""},
		{"every_12h", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := composePolicyToQuadlet(tt.input)
		if got != tt.want {
			t.Errorf("composePolicyToQuadlet(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
