package cooker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValueContainsRef_Exact(t *testing.T) {
	c := &Cooker{}
	if !c.valueContainsRef("db_data", "db_data", "cq-p-db_data") {
		t.Error("expected true for exact match")
	}
}

func TestValueContainsRef_SubstringNoMatch(t *testing.T) {
	c := &Cooker{}
	if c.valueContainsRef("cq-p-mariadb-nc_data", "db", "cq-p-db") {
		t.Error("expected false — 'db' is a substring of 'mariadb', not a standalone ref")
	}
}

func TestValueContainsRef_WithVolumeExt(t *testing.T) {
	c := &Cooker{}
	if !c.valueContainsRef("nc_data.volume", "nc_data", "cq-p-nc_data") {
		t.Error("expected true for volume ref with .volume suffix")
	}
}

func TestValueContainsRef_AlreadyRewritten(t *testing.T) {
	c := &Cooker{}
	if c.valueContainsRef("cq-p-nc_data", "nc_data", "cq-p-nc_data") {
		t.Error("expected false when value already contains newRef")
	}
}

func TestRewriteVolume_SubstringSafe(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, "nc_data.volume"), []byte("[Volume]\nName=nc_data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "db.container"), []byte("[Container]\nImage=postgres"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "nc.container"), []byte("[Container]\nImage=nextcloud\nVolume=nc_data.volume:/var/www/html"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "mariadb", false, 0, false)
	result, err := c.Cook()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ncContent := result.FileContents[filepath.Join(targetDir, "cq-mariadb-nc.container")]
	if strings.Contains(ncContent, "mariadb-db") {
		t.Errorf("volume reference should not contain 'mariadb-db' substring; got:\n%s", ncContent)
	}
	if !strings.Contains(ncContent, "cq-mariadb-nc_data.volume:/var/www/html") {
		t.Errorf("expected Volume=cq-mariadb-nc_data.volume:/var/www/html; got:\n%s", ncContent)
	}
}
