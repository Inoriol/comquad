package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Inoriol/comquad/internal/deploy"
)

// stateWithFiles builds a ProjectState whose Files are absolute paths formed
// by joining dir with each filename.
func stateWithFiles(dir string, names ...string) deploy.ProjectState {
	files := make([]string, len(names))
	for i, n := range names {
		files[i] = filepath.Join(dir, n)
	}
	return deploy.ProjectState{ProjectName: "myapp", Files: files}
}

// ---------------------------------------------------------------------------
// ContainerFileToUnitName
// ---------------------------------------------------------------------------

func TestContainerFileToUnitName_Basic(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/some/dir/cq-myapp-web.container", "cq-myapp-web.service"},
		{"/some/dir/cq-myapp-db.container", "cq-myapp-db.service"},
		{"cq-myapp-web.container", "cq-myapp-web.service"},
	}
	for _, c := range cases {
		got := ContainerFileToUnitName(c.in)
		if got != c.want {
			t.Errorf("ContainerFileToUnitName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// findComposeFile
// ---------------------------------------------------------------------------

func TestFindComposeFile_FindsComposeYaml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "compose.yaml"), "services: {}")
	got := findComposeFile(dir)
	if got != filepath.Join(dir, "compose.yaml") {
		t.Errorf("expected compose.yaml, got %q", got)
	}
}

func TestFindComposeFile_FindsDockerComposeYml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}")
	got := findComposeFile(dir)
	if got != filepath.Join(dir, "docker-compose.yml") {
		t.Errorf("expected docker-compose.yml, got %q", got)
	}
}

func TestFindComposeFile_PrefersComposeYamlOverDockerCompose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "compose.yaml"), "")
	writeFile(t, filepath.Join(dir, "docker-compose.yaml"), "")
	got := findComposeFile(dir)
	if got != filepath.Join(dir, "compose.yaml") {
		t.Errorf("expected compose.yaml to take precedence, got %q", got)
	}
}

func TestFindComposeFile_ReturnsEmptyWhenNotFound(t *testing.T) {
	dir := t.TempDir()
	got := findComposeFile(dir)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// MatchFirstContainer
// ---------------------------------------------------------------------------

func TestMatchFirstContainer_ByExactBaseName(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchFirstContainer("myapp", state, "cq-myapp-web.container")
	if got != filepath.Join(dir, "cq-myapp-web.container") {
		t.Errorf("unexpected match: %q", got)
	}
}

func TestMatchFirstContainer_ByNameWithoutExtension(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchFirstContainer("myapp", state, "cq-myapp-web")
	if got == "" {
		t.Error("expected match by name-without-extension")
	}
}

func TestMatchFirstContainer_ByServiceSuffix(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchFirstContainer("myapp", state, "cq-myapp-web.service")
	if got == "" {
		t.Error("expected match by .service suffix")
	}
}

func TestMatchFirstContainer_ByShortName(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchFirstContainer("myapp", state, "web")
	if got == "" {
		t.Error("expected match by short name (strip cq-myapp- prefix)")
	}
}

func TestMatchFirstContainer_ByCqPrefix(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchFirstContainer("myapp", state, "myapp-web")
	if got == "" {
		t.Error("expected match by stripping cq- prefix")
	}
}

func TestMatchFirstContainer_NoMatch(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchFirstContainer("myapp", state, "nonexistent")
	if got != "" {
		t.Errorf("expected no match, got %q", got)
	}
}

func TestMatchFirstContainer_SkipsNetworkFiles(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-default.network")
	got := MatchFirstContainer("myapp", state, "cq-myapp-default.network")
	if got != "" {
		t.Errorf("MatchContainer should not match .network files, got %q", got)
	}
}

func TestMatchFirstContainer_ByContainerName(t *testing.T) {
	dir := t.TempDir()
	containerFile := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerFile, "[Container]\nImage=nginx\nContainerName=myapp-web-custom\n")

	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchFirstContainer("myapp", state, "myapp-web-custom")
	if got == "" {
		t.Error("expected match by ContainerName= directive")
	}
	if got != containerFile {
		t.Errorf("expected %q, got %q", containerFile, got)
	}
}

func TestMatchFirstContainer_ByContainerNameNoMatch(t *testing.T) {
	dir := t.TempDir()
	containerFile := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerFile, "[Container]\nImage=nginx\nContainerName=myapp-web-custom\n")

	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchFirstContainer("myapp", state, "nonexistent")
	if got != "" {
		t.Errorf("expected no match, got %q", got)
	}
}

func TestReadContainerName_NoContainerName(t *testing.T) {
	dir := t.TempDir()
	containerFile := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerFile, "[Container]\nImage=nginx\n")

	got := readContainerName(containerFile)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestReadContainerName_WithContainerName(t *testing.T) {
	dir := t.TempDir()
	containerFile := filepath.Join(dir, "cq-myapp-web.container")
	writeFile(t, containerFile, "[Container]\nImage=nginx\nContainerName=myapp-web-custom\n")

	got := readContainerName(containerFile)
	if got != "myapp-web-custom" {
		t.Errorf("expected 'myapp-web-custom', got %q", got)
	}
}

func TestReadContainerName_MissingFile(t *testing.T) {
	got := readContainerName("/nonexistent/path/container.container")
	if got != "" {
		t.Errorf("expected empty string for missing file, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// MatchFirstContainers
// ---------------------------------------------------------------------------

func TestMatchAllContainers_ReturnsAllMatches(t *testing.T) {
	dir := t.TempDir()
	// Two services both named "web" from different pattern perspectives — not
	// realistic but tests the "return all" behaviour.
	state := stateWithFiles(dir, "cq-myapp-web.container", "cq-myapp-db.container")

	got := MatchAllContainers("myapp", state, "web")
	if len(got) != 1 {
		t.Errorf("expected 1 match for 'web', got %d: %v", len(got), got)
	}

	got = MatchAllContainers("myapp", state, "db")
	if len(got) != 1 {
		t.Errorf("expected 1 match for 'db', got %d: %v", len(got), got)
	}
}

func TestMatchAllContainers_EmptyOnNoMatch(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchAllContainers("myapp", state, "missing")
	if len(got) != 0 {
		t.Errorf("expected no matches, got %v", got)
	}
}

func TestMatchAllContainers_ConsistentWithMatchFirstContainer(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container", "cq-myapp-db.container")

	for _, arg := range []string{"web", "db", "cq-myapp-web", "cq-myapp-web.container"} {
		single := MatchFirstContainer("myapp", state, arg)
		multi := MatchAllContainers("myapp", state, arg)

		if single == "" && len(multi) > 0 {
			t.Errorf("arg=%q: MatchContainer found nothing but MatchContainers found %v", arg, multi)
		}
		if single != "" && len(multi) == 0 {
			t.Errorf("arg=%q: MatchContainer found %q but MatchContainers found nothing", arg, single)
		}
		if single != "" && len(multi) > 0 && multi[0] != single {
			t.Errorf("arg=%q: MatchContainer=%q, MatchContainers[0]=%q", arg, single, multi[0])
		}
	}
}

// ---------------------------------------------------------------------------
// MatchQuadletResource
// ---------------------------------------------------------------------------

func TestMatchQuadletResource_ByExactBaseName(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-default.network")
	got := MatchQuadletResource("myapp", state, "cq-myapp-default.network")
	if got == "" {
		t.Error("expected match by exact base name")
	}
}

func TestMatchQuadletResource_Volume(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-data.volume")
	got := MatchQuadletResource("myapp", state, "cq-myapp-data.volume")
	if got == "" {
		t.Error("expected match for .volume file")
	}
}

func TestMatchQuadletResource_NoMatchOnContainerFile(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchQuadletResource("myapp", state, "cq-myapp-web.container")
	if got != "" {
		t.Errorf("MatchQuadletResource should not match .container files, got %q", got)
	}
}

func TestMatchQuadletResource_NoMatch(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-default.network")
	got := MatchQuadletResource("myapp", state, "nonexistent")
	if got != "" {
		t.Errorf("expected no match, got %q", got)
	}
}

func TestMatchQuadletResource_ImageByServiceName(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.image")
	got := MatchQuadletResource("myapp", state, "cq-myapp-web-image.service")
	if got == "" {
		t.Error("expected match for .image via service unit name")
	}
}

func TestMatchQuadletResource_ImageByServiceNameWithoutExtension(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.image")
	got := MatchQuadletResource("myapp", state, "cq-myapp-web-image")
	if got == "" {
		t.Error("expected match for .image via service unit name without .service suffix")
	}
}

func TestMatchQuadletResource_ImageByBaseName(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.image")
	got := MatchQuadletResource("myapp", state, "cq-myapp-web.image")
	if got == "" {
		t.Error("expected match for .image via base filename")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile %q: %v", path, err)
	}
}
