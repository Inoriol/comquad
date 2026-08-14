package reconcile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
	"github.com/Inoriol/comquad/compose2quadlet/serialization"
)

func container(name string, secs ...c2q.Section) c2q.QuadletUnit {
	return c2q.QuadletUnit{Type: c2q.UnitContainer, Name: name, Sections: secs}
}

func sec(name string, dirs ...c2q.Directive) c2q.Section {
	return c2q.Section{Name: name, Directives: dirs}
}

func dir(key string, vals ...string) c2q.Directive {
	return c2q.Directive{Key: key, Values: vals}
}

func TestMergeUnit_Unchanged(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	disk := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	new := container("cq-app-web", sec("Container", dir("Image", "web.image")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	assertUnitEqual(t, merged, base)
}

func TestMergeUnit_UserChangedOnly(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	disk := container("cq-app-web", sec("Container", dir("Image", "custom.image")))
	new := container("cq-app-web", sec("Container", dir("Image", "web.image")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	assertUnitEqual(t, merged, disk)
}

func TestMergeUnit_ComposeChangedOnly(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	disk := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	new := container("cq-app-web", sec("Container", dir("Image", "web-v2.image")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	assertUnitEqual(t, merged, new)
}

func TestMergeUnit_BothChangedSame(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	disk := container("cq-app-web", sec("Container", dir("Image", "shared.image")))
	new := container("cq-app-web", sec("Container", dir("Image", "shared.image")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	assertUnitEqual(t, merged, new)
}

func TestMergeUnit_Conflict_UserWins(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	disk := container("cq-app-web", sec("Container", dir("Image", "custom.image")))
	new := container("cq-app-web", sec("Container", dir("Image", "web-v2.image")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %v", conflicts)
	}
	c := conflicts[0]
	if c.Key != "Image" || c.User != "custom.image" || c.Generated != "web-v2.image" {
		t.Fatalf("unexpected conflict: %+v", c)
	}
	assertUnitEqual(t, merged, disk)
}

func TestMergeUnit_UserAdded(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	disk := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("Volume", "/data")))
	new := container("cq-app-web", sec("Container", dir("Image", "web.image")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	assertUnitEqual(t, merged, disk)
}

func TestMergeUnit_ComposeAdded(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	disk := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	new := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("PublishPort", "8080:80")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	assertUnitEqual(t, merged, new)
}

func TestMergeUnit_UserRemoved(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("PublishPort", "8080:80")))
	disk := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	new := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("PublishPort", "8080:80")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	assertUnitEqual(t, merged, disk)
}

func TestMergeUnit_ComposeRemoved(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("PublishPort", "8080:80")))
	disk := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("PublishPort", "8080:80")))
	new := container("cq-app-web", sec("Container", dir("Image", "web.image")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	assertUnitEqual(t, merged, new)
}

func TestMergeUnit_UserChangedComposeRemoved(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("PublishPort", "8080:80")))
	disk := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("PublishPort", "9090:80")))
	new := container("cq-app-web", sec("Container", dir("Image", "web.image")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %v", conflicts)
	}
	if conflicts[0].Generated != removedSentinel {
		t.Fatalf("expected removed sentinel, got %q", conflicts[0].Generated)
	}
	assertUnitEqual(t, merged, disk)
}

func TestMergeUnit_UserRemovedComposeChanged(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("PublishPort", "8080:80")))
	disk := container("cq-app-web", sec("Container", dir("Image", "web.image")))
	new := container("cq-app-web", sec("Container", dir("Image", "web.image"), dir("PublishPort", "9090:80")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %v", conflicts)
	}
	if conflicts[0].User != removedSentinel {
		t.Fatalf("expected removed sentinel, got %q", conflicts[0].User)
	}
	assertUnitEqual(t, merged, disk)
}

func TestMergeUnit_MultiValueConflict(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("Environment", "A=1", "B=2")))
	disk := container("cq-app-web", sec("Container", dir("Environment", "A=1", "B=2", "C=3")))
	new := container("cq-app-web", sec("Container", dir("Environment", "A=1", "B=2", "D=4")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %v", conflicts)
	}
	// Key-level granularity: user wins, so compose-added D is dropped but reported.
	assertUnitEqual(t, merged, disk)
}

func TestMergeUnit_BooleanDirective(t *testing.T) {
	base := container("cq-app-web", sec("Container", dir("ReadOnly", "true"), dir("NoNewPrivileges")))
	disk := container("cq-app-web", sec("Container", dir("ReadOnly", "true"), dir("NoNewPrivileges")))
	new := container("cq-app-web", sec("Container", dir("ReadOnly", "true"), dir("NoNewPrivileges")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	assertUnitEqual(t, merged, base)
}

func TestMergeUnit_SectionOrdering(t *testing.T) {
	base := container("cq-app-web", sec("Unit", dir("After", "db.container")), sec("Container", dir("Image", "web.image")))
	disk := container("cq-app-web", sec("Unit", dir("After", "db.container")), sec("Container", dir("Image", "web.image")))
	new := container("cq-app-web", sec("Unit", dir("After", "db.container")), sec("Container", dir("Image", "web.image")))

	merged, conflicts := MergeUnit(base, disk, new)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	if len(merged.Sections) != 2 || merged.Sections[0].Name != "Unit" || merged.Sections[1].Name != "Container" {
		t.Fatalf("unexpected section order: %+v", merged.Sections)
	}
}

func assertUnitEqual(t *testing.T, got, want c2q.QuadletUnit) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unit mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func TestReconcile_FirstDeploy(t *testing.T) {
	target := t.TempDir()
	baseline := t.TempDir()
	units := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web.image")))}

	res, err := Reconcile(target, baseline, "cq-app-", units)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 1 || len(res.Changed) != 0 || len(res.Removed) != 0 || len(res.Conflicts) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	assertFileContains(t, filepath.Join(target, "cq-app-web.container"), "Image=web.image")
	assertFileContains(t, filepath.Join(baseline, "cq-app-web.container"), "Image=web.image")
}

func TestReconcile_ReDeployUnchanged(t *testing.T) {
	target := t.TempDir()
	baseline := t.TempDir()
	units := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web.image")))}

	if _, err := Reconcile(target, baseline, "cq-app-", units); err != nil {
		t.Fatal(err)
	}
	res, err := Reconcile(target, baseline, "cq-app-", units)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 0 || len(res.Changed) != 0 || len(res.Removed) != 0 || len(res.Conflicts) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestReconcile_ComposeChange(t *testing.T) {
	target := t.TempDir()
	baseline := t.TempDir()
	v1 := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web.image")))}
	v2 := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web-v2.image")))}

	if _, err := Reconcile(target, baseline, "cq-app-", v1); err != nil {
		t.Fatal(err)
	}
	res, err := Reconcile(target, baseline, "cq-app-", v2)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Changed) != 1 || len(res.Conflicts) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	assertFileContains(t, filepath.Join(target, "cq-app-web.container"), "Image=web-v2.image")
}

func TestReconcile_PreservesManualEdit(t *testing.T) {
	target := t.TempDir()
	baseline := t.TempDir()
	units := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web.image")))}

	if _, err := Reconcile(target, baseline, "cq-app-", units); err != nil {
		t.Fatal(err)
	}

	// Simulate a manual edit: change Image= on disk only.
	disk := container("cq-app-web", sec("Container", dir("Image", "custom.image")))
	writeDisk(t, target, disk)

	// Re-deploy with unchanged compose: the manual edit must be preserved.
	res, err := Reconcile(target, baseline, "cq-app-", units)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Changed) != 0 || len(res.Conflicts) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	assertFileContains(t, filepath.Join(target, "cq-app-web.container"), "Image=custom.image")
}

func TestReconcile_ConflictPreservesUserEdit(t *testing.T) {
	target := t.TempDir()
	baseline := t.TempDir()
	v1 := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web.image")))}
	v2 := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web-v2.image")))}

	if _, err := Reconcile(target, baseline, "cq-app-", v1); err != nil {
		t.Fatal(err)
	}

	disk := container("cq-app-web", sec("Container", dir("Image", "custom.image")))
	writeDisk(t, target, disk)

	res, err := Reconcile(target, baseline, "cq-app-", v2)
	if err != nil {
		t.Fatal(err)
	}
	// User wins the conflict, so the file is left unchanged (no rewrite) but the
	// conflict is reported for visibility.
	if len(res.Changed) != 0 || len(res.Conflicts) != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
	assertFileContains(t, filepath.Join(target, "cq-app-web.container"), "Image=custom.image")
}

func TestReconcile_NoBaselineFallback(t *testing.T) {
	target := t.TempDir()
	baseline := t.TempDir()

	// Pre-seed a target file with no matching baseline (e.g. after regenerate).
	disk := container("cq-app-web", sec("Container", dir("Image", "custom.image")))
	writeDisk(t, target, disk)

	units := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web.image")))}
	res, err := Reconcile(target, baseline, "cq-app-", units)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Changed) != 1 || len(res.NoBaseline) != 1 || len(res.Conflicts) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	assertFileContains(t, filepath.Join(target, "cq-app-web.container"), "Image=web.image")
}

func TestReconcile_RemovesStaleFiles(t *testing.T) {
	target := t.TempDir()
	baseline := t.TempDir()

	old := []c2q.QuadletUnit{
		container("cq-app-web", sec("Container", dir("Image", "web.image"))),
		container("cq-app-db", sec("Container", dir("Image", "db.image"))),
	}
	if _, err := Reconcile(target, baseline, "cq-app-", old); err != nil {
		t.Fatal(err)
	}

	// db service removed from compose.
	newUnits := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web.image")))}
	res, err := Reconcile(target, baseline, "cq-app-", newUnits)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) != 1 {
		t.Fatalf("expected 1 removed file, got %+v", res.Removed)
	}
	if _, err := os.Stat(filepath.Join(target, "cq-app-db.container")); !os.IsNotExist(err) {
		t.Fatal("expected stale db.container to be removed")
	}
	if _, err := os.Stat(filepath.Join(baseline, "cq-app-db.container")); !os.IsNotExist(err) {
		t.Fatal("expected stale baseline db.container to be removed")
	}
}

func TestReconcile_AddsNewService(t *testing.T) {
	target := t.TempDir()
	baseline := t.TempDir()

	v1 := []c2q.QuadletUnit{container("cq-app-web", sec("Container", dir("Image", "web.image")))}
	if _, err := Reconcile(target, baseline, "cq-app-", v1); err != nil {
		t.Fatal(err)
	}

	v2 := []c2q.QuadletUnit{
		container("cq-app-web", sec("Container", dir("Image", "web.image"))),
		container("cq-app-db", sec("Container", dir("Image", "db.image"))),
	}
	res, err := Reconcile(target, baseline, "cq-app-", v2)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 1 {
		t.Fatalf("expected 1 created file, got %+v", res.Created)
	}
	assertFileContains(t, filepath.Join(target, "cq-app-db.container"), "Image=db.image")
}

func writeDisk(t *testing.T, target string, units ...c2q.QuadletUnit) {
	t.Helper()
	for _, u := range units {
		filename := u.Name + "." + string(u.Type)
		if err := writeFileAtomic(filepath.Join(target, filename), serialization.Marshal(u)); err != nil {
			t.Fatal(err)
		}
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("file %s: got %q, want substring %q", path, data, want)
	}
}
