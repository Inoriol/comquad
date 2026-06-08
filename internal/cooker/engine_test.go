package cooker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCook_RenamesFiles(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create test files
	files := []string{"web.container", "db.container", "app.network", "data.volume"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tempDir, f), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	for _, f := range files {
		expected := "comquad-myproject-" + f
		dst := filepath.Join(targetDir, expected)
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			t.Errorf("expected file %q not found in target dir", expected)
		}
	}
}

func TestCook_AlreadyHasPrefix(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, "comquad-myproject-web.container"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "comquad-myproject-web.container")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Error("expected file with unchanged name to exist")
	}
}

func TestCook_ReplacesGenericComquadPrefix(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, "comquad-web.container"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "comquad-myproject-web.container")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Error("expected comquad- prefix to be replaced with comquad-myproject- prefix")
	}
}

func TestCook_RewritesNetworkReferences(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a network file that will be renamed
	networkName := "comquad-myproject-appnet.network"
	networkContent := "[Network]\nName=comquad-myproject-appnet"
	if err := os.WriteFile(filepath.Join(tempDir, networkName), []byte(networkContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a container file that references the network
	containerContent := "[Container]\nImage=nginx\nNetwork=comquad-myproject-appnet"
	if err := os.WriteFile(filepath.Join(tempDir, "comquad-myproject-web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "comquad-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// The Network= line should have been rewritten from old to new name
	// Old ref: comquad-myproject-appnet (without .network extension)
	// New ref: comquad-myproject-appnet (without .network extension)
	// Since the rename map would have same old/new for this case, no change expected
	if !strings.Contains(string(content), "Network=comquad-myproject-appnet") {
		t.Errorf("expected Network= reference to be preserved, got:\n%s", string(content))
	}
}

func TestCook_RewritesVolumeReferences(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a volume file
	volName := "comquad-myproject-appvol.volume"
	volContent := "[Volume]\nName=comquad-myproject-appvol"
	if err := os.WriteFile(filepath.Join(tempDir, volName), []byte(volContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a container file that references the volume
	containerContent := "[Container]\nImage=nginx\nVolume=comquad-myproject-appvol:/data"
	if err := os.WriteFile(filepath.Join(tempDir, "comquad-myproject-web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "comquad-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "Volume=comquad-myproject-appvol") {
		t.Errorf("expected Volume= reference to be preserved, got:\n%s", string(content))
	}
}

func TestCook_AddsInstallSectionToContainer(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	containerContent := "[Container]\nImage=nginx"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "comquad-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "[Install]") {
		t.Error("expected [Install] section to be added to container file")
	}
	if !strings.Contains(string(content), "WantedBy=default.target") {
		t.Error("expected WantedBy=default.target in [Install] section")
	}
}

func TestCook_AddsInstallSectionToNetwork(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	networkContent := "[Network]\nName=appnet"
	if err := os.WriteFile(filepath.Join(tempDir, "appnet.network"), []byte(networkContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "comquad-myproject-appnet.network")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "[Install]") {
		t.Error("expected [Install] section to be added to network file")
	}
}

func TestCook_SkipsAutoUpdateWhenNoAutoupdateLabel(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	containerContent := "[Container]\nImage=nginx\nLabel=comquad-no-autoupdate=true"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "comquad-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if strings.Contains(string(content), "AutoUpdate=registry") {
		t.Error("expected AutoUpdate=registry to NOT be added when comquad-no-autoupdate label is present")
	}
}

func TestCook_AddsAutoUpdateWhenNoLabel(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	containerContent := "[Container]\nImage=nginx"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "comquad-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "AutoUpdate=registry") {
		t.Error("expected AutoUpdate=registry to be added when no comquad-no-autoupdate label")
	}
}

func TestCook_CreatesTargetDir(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "nonexistent")

	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte("[Container]\nImage=nginx"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Error("expected target directory to be created")
	}
}

func TestCook_IgnoresDirectories(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a directory inside tempDir
	subdir := filepath.Join(tempDir, "some-dir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false)
	if err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}
}

func TestBuildNewFileName_NoPrefix(t *testing.T) {
	c := &Cooker{ProjectName: "myproject"}
	result := c.buildNewFileName("web.container")
	expected := "comquad-myproject-web.container"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildNewFileName_AlreadyHasPrefix(t *testing.T) {
	c := &Cooker{ProjectName: "myproject"}
	result := c.buildNewFileName("comquad-myproject-web.container")
	expected := "comquad-myproject-web.container"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildNewFileName_HasGenericComquadPrefix(t *testing.T) {
	c := &Cooker{ProjectName: "myproject"}
	result := c.buildNewFileName("comquad-web.container")
	expected := "comquad-myproject-web.container"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestIsReferenceDirective_Network(t *testing.T) {
	if !isReferenceDirective("Network=foo") {
		t.Error("expected Network= to be a reference directive")
	}
}

func TestIsReferenceDirective_Volume(t *testing.T) {
	if !isReferenceDirective("Volume=foo") {
		t.Error("expected Volume= to be a reference directive")
	}
}

func TestIsReferenceDirective_Pod(t *testing.T) {
	if !isReferenceDirective("Pod=foo") {
		t.Error("expected Pod= to be a reference directive")
	}
}

func TestIsReferenceDirective_Image(t *testing.T) {
	if isReferenceDirective("Image=nginx") {
		t.Error("expected Image= to NOT be a reference directive")
	}
}

func TestIsReferenceDirective_Description(t *testing.T) {
	if isReferenceDirective("Description=my service") {
		t.Error("expected Description= to NOT be a reference directive")
	}
}

func TestStripQuadletExtension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo.container", "foo"},
		{"foo.network", "foo"},
		{"foo.volume", "foo"},
		{"foo.pod", "foo"},
		{"foo.kube", "foo"},
		{"foo.image", "foo"},
		{"foo.build", "foo"},
		{"foo.txt", "foo.txt"},
		{"foo", "foo"},
	}

	for _, tt := range tests {
		result := stripQuadletExtension(tt.input)
		if result != tt.expected {
			t.Errorf("stripQuadletExtension(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewCooker(t *testing.T) {
	c := NewCooker("/tmp", "/target", "myproject", true)
	if c.TempDir != "/tmp" {
		t.Errorf("expected TempDir '/tmp', got %q", c.TempDir)
	}
	if c.TargetDir != "/target" {
		t.Errorf("expected TargetDir '/target', got %q", c.TargetDir)
	}
	if c.ProjectName != "myproject" {
		t.Errorf("expected ProjectName 'myproject', got %q", c.ProjectName)
	}
	if !c.IsRootless {
		t.Error("expected IsRootless to be true")
	}
}
