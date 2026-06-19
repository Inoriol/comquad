package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"comquad/internal/deploy"
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
// MatchContainer
// ---------------------------------------------------------------------------

func TestMatchContainer_ByExactBaseName(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchContainer("myapp", state, "cq-myapp-web.container")
	if got != filepath.Join(dir, "cq-myapp-web.container") {
		t.Errorf("unexpected match: %q", got)
	}
}

func TestMatchContainer_ByNameWithoutExtension(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchContainer("myapp", state, "cq-myapp-web")
	if got == "" {
		t.Error("expected match by name-without-extension")
	}
}

func TestMatchContainer_ByServiceSuffix(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchContainer("myapp", state, "cq-myapp-web.service")
	if got == "" {
		t.Error("expected match by .service suffix")
	}
}

func TestMatchContainer_ByShortName(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchContainer("myapp", state, "web")
	if got == "" {
		t.Error("expected match by short name (strip cq-myapp- prefix)")
	}
}

func TestMatchContainer_ByCqPrefix(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchContainer("myapp", state, "myapp-web")
	if got == "" {
		t.Error("expected match by stripping cq- prefix")
	}
}

func TestMatchContainer_NoMatch(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchContainer("myapp", state, "nonexistent")
	if got != "" {
		t.Errorf("expected no match, got %q", got)
	}
}

func TestMatchContainer_SkipsNetworkFiles(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-default.network")
	got := MatchContainer("myapp", state, "cq-myapp-default.network")
	if got != "" {
		t.Errorf("MatchContainer should not match .network files, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// MatchContainers
// ---------------------------------------------------------------------------

func TestMatchContainers_ReturnsAllMatches(t *testing.T) {
	dir := t.TempDir()
	// Two services both named "web" from different pattern perspectives — not
	// realistic but tests the "return all" behaviour.
	state := stateWithFiles(dir, "cq-myapp-web.container", "cq-myapp-db.container")

	got := MatchContainers("myapp", state, "web")
	if len(got) != 1 {
		t.Errorf("expected 1 match for 'web', got %d: %v", len(got), got)
	}

	got = MatchContainers("myapp", state, "db")
	if len(got) != 1 {
		t.Errorf("expected 1 match for 'db', got %d: %v", len(got), got)
	}
}

func TestMatchContainers_EmptyOnNoMatch(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchContainers("myapp", state, "missing")
	if len(got) != 0 {
		t.Errorf("expected no matches, got %v", got)
	}
}

func TestMatchContainers_ConsistentWithMatchContainer(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container", "cq-myapp-db.container")

	for _, arg := range []string{"web", "db", "cq-myapp-web", "cq-myapp-web.container"} {
		single := MatchContainer("myapp", state, arg)
		multi := MatchContainers("myapp", state, arg)

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
// MatchNetworkOrVolume
// ---------------------------------------------------------------------------

func TestMatchNetworkOrVolume_ByExactBaseName(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-default.network")
	got := MatchNetworkOrVolume("myapp", state, "cq-myapp-default.network")
	if got == "" {
		t.Error("expected match by exact base name")
	}
}

func TestMatchNetworkOrVolume_Volume(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-data.volume")
	got := MatchNetworkOrVolume("myapp", state, "cq-myapp-data.volume")
	if got == "" {
		t.Error("expected match for .volume file")
	}
}

func TestMatchNetworkOrVolume_NoMatchOnContainerFile(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-web.container")
	got := MatchNetworkOrVolume("myapp", state, "cq-myapp-web.container")
	if got != "" {
		t.Errorf("MatchNetworkOrVolume should not match .container files, got %q", got)
	}
}

func TestMatchNetworkOrVolume_NoMatch(t *testing.T) {
	dir := t.TempDir()
	state := stateWithFiles(dir, "cq-myapp-default.network")
	got := MatchNetworkOrVolume("myapp", state, "nonexistent")
	if got != "" {
		t.Errorf("expected no match, got %q", got)
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
