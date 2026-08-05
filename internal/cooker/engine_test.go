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

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	for _, f := range files {
		expected := "cq-myproject-" + f
		dst := filepath.Join(targetDir, expected)
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			t.Errorf("expected file %q not found in target dir", expected)
		}
	}
}

func TestCook_AlreadyHasPrefix(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, "cq-myproject-web.container"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Error("expected file with unchanged name to exist")
	}
}

func TestCook_ReplacesGenericComquadPrefix(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, "cq-web.container"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Error("expected cq- prefix to be replaced with cq-myproject- prefix")
	}
}

func TestCook_RewritesNetworkReferences(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a network file that will be renamed
	networkName := "cq-myproject-appnet.network"
	networkContent := "[Network]\nName=cq-myproject-appnet"
	if err := os.WriteFile(filepath.Join(tempDir, networkName), []byte(networkContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a container file that references the network
	containerContent := "[Container]\nImage=nginx\nNetwork=cq-myproject-appnet"
	if err := os.WriteFile(filepath.Join(tempDir, "cq-myproject-web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// The Network= line should have been rewritten from old to new name
	// Old ref: cq-myproject-appnet (without .network extension)
	// New ref: cq-myproject-appnet (without .network extension)
	// Since the rename map would have same old/new for this case, no change expected
	if !strings.Contains(string(content), "Network=cq-myproject-appnet") {
		t.Errorf("expected Network= reference to be preserved, got:\n%s", string(content))
	}
}

func TestCook_RewritesVolumeReferences(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a volume file
	volName := "cq-myproject-appvol.volume"
	volContent := "[Volume]\nName=cq-myproject-appvol"
	if err := os.WriteFile(filepath.Join(tempDir, volName), []byte(volContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a container file that references the volume
	containerContent := "[Container]\nImage=nginx\nVolume=cq-myproject-appvol:/data"
	if err := os.WriteFile(filepath.Join(tempDir, "cq-myproject-web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "Volume=cq-myproject-appvol") {
		t.Errorf("expected Volume= reference to be preserved, got:\n%s", string(content))
	}
}

func TestCook_NoDoublePrefixOnNetworkReference(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a network file with the cq- prefix already added
	if err := os.WriteFile(filepath.Join(tempDir, "dbnet.network"), []byte("[Network]\nName=dbnet"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a container file that already references the prefixed network name
	// This simulates the case where podlet or a previous pass already added the prefix
	containerContent := "[Container]\nImage=nginx\nNetwork=cq-myproject-dbnet.network"
	if err := os.WriteFile(filepath.Join(tempDir, "db.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-db.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if strings.Contains(string(content), "cq-myproject-cq-myproject-") {
		t.Errorf("double prefix detected in Network=:\n%s", string(content))
	}
	if !strings.Contains(string(content), "Network=cq-myproject-dbnet.network") {
		t.Errorf("expected Network=cq-myproject-dbnet.network, got:\n%s", string(content))
	}
}

func TestCook_NoDoublePrefixOnVolumeReference(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a volume file without prefix
	if err := os.WriteFile(filepath.Join(tempDir, "db_data.volume"), []byte("[Volume]\nName=db_data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a container file that already references the prefixed volume name
	containerContent := "[Container]\nImage=nginx\nVolume=cq-myproject-db_data.volume:/var/lib/mysql"
	if err := os.WriteFile(filepath.Join(tempDir, "db.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-db.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if strings.Contains(string(content), "cq-myproject-cq-myproject-") {
		t.Errorf("double prefix detected in Volume=:\n%s", string(content))
	}
	if !strings.Contains(string(content), "Volume=cq-myproject-db_data.volume:/var/lib/mysql") {
		t.Errorf("expected Volume=cq-myproject-db_data.volume:/var/lib/mysql, got:\n%s", string(content))
	}
}

func TestCook_RewritesUnitSectionAfter(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a container file that will be renamed (web -> cq-myproject-web)
	webContent := "[Container]\nImage=nginx"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(webContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a container file that references web in [Unit] section with .service suffix
	apiContent := "[Unit]\nAfter=web.service\nRequires=web.service\n\n[Container]\nImage=node"
	if err := os.WriteFile(filepath.Join(tempDir, "api.container"), []byte(apiContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-api.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "After=cq-myproject-web.service") {
		t.Errorf("expected After=cq-myproject-web.service, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "Requires=cq-myproject-web.service") {
		t.Errorf("expected Requires=cq-myproject-web.service, got:\n%s", string(content))
	}
}

func TestCook_RewritesUnitSectionMultipleRefs(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create two container files that will be renamed
	for _, name := range []string{"web", "db"} {
		content := "[Container]\nImage=nginx"
		if err := os.WriteFile(filepath.Join(tempDir, name+".container"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a container file that references both in [Unit] section
	cacheContent := "[Unit]\nAfter=web.service db.service\n\n[Container]\nImage=redis"
	if err := os.WriteFile(filepath.Join(tempDir, "cache.container"), []byte(cacheContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-cache.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "After=cq-myproject-web.service cq-myproject-db.service") {
		t.Errorf("expected both references rewritten, got:\n%s", string(content))
	}
}

func TestCook_NoDoublePrefixOnUnitReference(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a container file without prefix
	if err := os.WriteFile(filepath.Join(tempDir, "db.container"), []byte("[Container]\nImage=postgres"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a container file that already references the prefixed name
	containerContent := "[Unit]\nAfter=cq-myproject-db.service\n\n[Container]\nImage=nginx"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if strings.Contains(string(content), "cq-myproject-cq-myproject-") {
		t.Errorf("double prefix detected in After=:\n%s", string(content))
	}
	if !strings.Contains(string(content), "After=cq-myproject-db.service") {
		t.Errorf("expected After=cq-myproject-db.service, got:\n%s", string(content))
	}
}

func TestCook_RewritesUnitSectionWithCqPrefix(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a container file with cq- prefix (podlet behavior)
	webContent := "[Container]\nImage=nginx"
	if err := os.WriteFile(filepath.Join(tempDir, "cq-web.container"), []byte(webContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a container file that references cq-web in [Unit] section
	apiContent := "[Unit]\nAfter=cq-web.service\nRequires=cq-web.service\n\n[Container]\nImage=node"
	if err := os.WriteFile(filepath.Join(tempDir, "cq-api.container"), []byte(apiContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-api.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "After=cq-myproject-web.service") {
		t.Errorf("expected After=cq-myproject-web.service, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "Requires=cq-myproject-web.service") {
		t.Errorf("expected Requires=cq-myproject-web.service, got:\n%s", string(content))
	}
}

func TestCook_AddsInstallSectionToContainer(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	containerContent := "[Container]\nImage=nginx"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
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

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-appnet.network")
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

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
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

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
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

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
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

	c := NewCooker(tempDir, targetDir, "myproject", false, 0, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}
}

func TestBuildNewFileName_NoPrefix(t *testing.T) {
	c := &Cooker{ProjectName: "myproject"}
	result := c.buildNewFileName("web.container")
	expected := "cq-myproject-web.container"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildNewFileName_AlreadyHasPrefix(t *testing.T) {
	c := &Cooker{ProjectName: "myproject"}
	result := c.buildNewFileName("cq-myproject-web.container")
	expected := "cq-myproject-web.container"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildNewFileName_HasGenericComquadPrefix(t *testing.T) {
	c := &Cooker{ProjectName: "myproject"}
	result := c.buildNewFileName("cq-web.container")
	expected := "cq-myproject-web.container"
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

func TestIsReferenceDirective_After(t *testing.T) {
	if !isReferenceDirective("After=foo") {
		t.Error("expected After= to be a reference directive")
	}
}

func TestIsReferenceDirective_Requires(t *testing.T) {
	if !isReferenceDirective("Requires=foo") {
		t.Error("expected Requires= to be a reference directive")
	}
}

func TestIsReferenceDirective_Conflicts(t *testing.T) {
	if !isReferenceDirective("Conflicts=foo") {
		t.Error("expected Conflicts= to be a reference directive")
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
		{"foo.service", "foo"},
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
	c := NewCooker("/tmp", "/target", "myproject", true, 0, false)
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
	if c.SELinuxEnabled {
		t.Error("expected SELinuxEnabled to be false")
	}
}

func TestParseHostPort_Basic(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"80", 80, false},
		{"8080", 8080, false},
		{"443/tcp", 443, false},
		{"8080/udp", 8080, false},
		{"80:80", 80, false},
		{"8080:80", 8080, false},
		{"127.0.0.1:80:80", 80, false},
		{"192.168.1.1:443:443/tcp", 443, false},
		{"invalid", 0, true},
		{"abc", 0, true},
		{":", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseHostPort(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseHostPort(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseHostPort(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("parseHostPort(%q) = %d, want %d", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestRebuildPublishPort_Basic(t *testing.T) {
	c := &Cooker{}

	tests := []struct {
		portStr   string
		newPort   int
		expected  string
	}{
		{"80", 8080, "8080"},
		{"80:80", 8080, "8080:80"},
		{"80:443", 8080, "8080:443"},
		{"127.0.0.1:80:80", 8080, "127.0.0.1:8080:80"},
		{"80/tcp", 8080, "8080/tcp"},
		{"80:80/tcp", 8080, "8080:80/tcp"},
		{"80:80/udp", 8080, "8080:80/udp"},
		{"127.0.0.1:80:80/udp", 8080, "127.0.0.1:8080:80/udp"},
	}

	for _, tt := range tests {
		t.Run(tt.portStr, func(t *testing.T) {
			result := c.rebuildPublishPort(tt.portStr, tt.newPort)
			if result != tt.expected {
				t.Errorf("rebuildPublishPort(%q, %d) = %q, want %q", tt.portStr, tt.newPort, result, tt.expected)
			}
		})
	}
}

func TestCook_PortOffsetting_PrivilegedPorts(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	containerContent := "[Container]\nImage=nginx\nPublishPort=80\nPublishPort=443/tcp"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", true, 2000, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Port 80 should be offset to 2080
	if !strings.Contains(string(content), "PublishPort=2080") {
		t.Errorf("expected PublishPort=2080, got:\n%s", string(content))
	}
	// Port 443 should be offset to 2443 (443 + 2000), but 2043 is claimed by another port so it increments
	if !strings.Contains(string(content), "PublishPort=2443/tcp") {
		t.Errorf("expected PublishPort=2443/tcp, got:\n%s", string(content))
	}
}

func TestCook_PortOffsetting_UnprivilegedPorts(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	containerContent := "[Container]\nImage=nginx\nPublishPort=8080"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", true, 2000, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Port 8080 should remain unchanged (>= 1024)
	if !strings.Contains(string(content), "PublishPort=8080") {
		t.Errorf("expected PublishPort=8080 unchanged, got:\n%s", string(content))
	}
}

func TestCook_PortOffsetting_NoOffsetForNonRootless(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	containerContent := "[Container]\nImage=nginx\nPublishPort=80"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(containerContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", false, 2000, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	dst := filepath.Join(targetDir, "cq-myproject-web.container")
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Port 80 should remain unchanged when not rootless
	if !strings.Contains(string(content), "PublishPort=80") {
		t.Errorf("expected PublishPort=80 unchanged in non-rootless mode, got:\n%s", string(content))
	}
}



func TestCook_PortOffsetting_InternalConflict(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Two services with privileged ports that would conflict after offsetting
	// web: PublishPort=80 -> 2080
	// api: PublishPort=81 -> 2081 (no conflict)
	webContent := "[Container]\nImage=nginx\nPublishPort=80"
	apiContent := "[Container]\nImage=node\nPublishPort=81"
	if err := os.WriteFile(filepath.Join(tempDir, "web.container"), []byte(webContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "api.container"), []byte(apiContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewCooker(tempDir, targetDir, "myproject", true, 2000, false)
	if _, err := c.Cook(); err != nil {
		t.Fatalf("Cook failed: %v", err)
	}

	webDst := filepath.Join(targetDir, "cq-myproject-web.container")
	webOut, err := os.ReadFile(webDst)
	if err != nil {
		t.Fatalf("failed to read web output: %v", err)
	}

	apiDst := filepath.Join(targetDir, "cq-myproject-api.container")
	apiOut, err := os.ReadFile(apiDst)
	if err != nil {
		t.Fatalf("failed to read api output: %v", err)
	}

	if !strings.Contains(string(webOut), "PublishPort=2080") {
		t.Errorf("web: expected PublishPort=2080, got:\n%s", string(webOut))
	}
	if !strings.Contains(string(apiOut), "PublishPort=2081") {
		t.Errorf("api: expected PublishPort=2081, got:\n%s", string(apiOut))
	}
}

func TestAddSELinuxLabels_NoOptions(t *testing.T) {
	content := "[Container]\nImage=nginx\nVolume=appvol:/data"
	c := &Cooker{SELinuxEnabled: true}
	result := c.addSELinuxLabels(content)

	if !strings.Contains(result, "Volume=appvol:/data:z") {
		t.Errorf("expected :z appended, got:\n%s", result)
	}
}

func TestAddSELinuxLabels_RoOption(t *testing.T) {
	content := "[Container]\nImage=nginx\nVolume=appvol:/data:ro"
	c := &Cooker{SELinuxEnabled: true}
	result := c.addSELinuxLabels(content)

	if !strings.Contains(result, "Volume=appvol:/data:ro,z") {
		t.Errorf("expected :ro,z appended, got:\n%s", result)
	}
}

func TestAddSELinuxLabels_RwOption(t *testing.T) {
	content := "[Container]\nImage=nginx\nVolume=appvol:/data:rw"
	c := &Cooker{SELinuxEnabled: true}
	result := c.addSELinuxLabels(content)

	if !strings.Contains(result, "Volume=appvol:/data:rw,z") {
		t.Errorf("expected :rw,z appended, got:\n%s", result)
	}
}

func TestAddSELinuxLabels_AlreadyHasZ(t *testing.T) {
	content := "[Container]\nImage=nginx\nVolume=appvol:/data:z"
	c := &Cooker{SELinuxEnabled: true}
	result := c.addSELinuxLabels(content)

	if !strings.Contains(result, "Volume=appvol:/data:z") {
		t.Errorf("expected :z preserved without duplication, got:\n%s", result)
	}
	if strings.Contains(result, "z,z") {
		t.Errorf("expected no double :z, got:\n%s", result)
	}
}

func TestAddSELinuxLabels_AlreadyHasZUpper(t *testing.T) {
	content := "[Container]\nImage=nginx\nVolume=appvol:/data:Z"
	c := &Cooker{SELinuxEnabled: true}
	result := c.addSELinuxLabels(content)

	if !strings.Contains(result, "Volume=appvol:/data:Z") {
		t.Errorf("expected :Z preserved without modification, got:\n%s", result)
	}
}

func TestAddSELinuxLabels_Disabled(t *testing.T) {
	content := "[Container]\nImage=nginx\nVolume=appvol:/data"
	c := &Cooker{SELinuxEnabled: false}
	result := c.addSELinuxLabels(content)

	if !strings.Contains(result, "Volume=appvol:/data") {
		t.Errorf("expected no modification when SELinux disabled, got:\n%s", result)
	}
	if strings.Contains(result, ":z") {
		t.Errorf("expected no :z added when SELinux disabled, got:\n%s", result)
	}
}

func TestAddSELinuxLabels_BindMount(t *testing.T) {
	content := "[Container]\nImage=nginx\nVolume=/host/path:/container/path"
	c := &Cooker{SELinuxEnabled: true}
	result := c.addSELinuxLabels(content)

	if !strings.Contains(result, "Volume=/host/path:/container/path:z") {
		t.Errorf("expected :z appended to bind mount, got:\n%s", result)
	}
}

func TestAddSELinuxLabels_BindMountRo(t *testing.T) {
	content := "[Container]\nImage=nginx\nVolume=/host/path:/container/path:ro"
	c := &Cooker{SELinuxEnabled: true}
	result := c.addSELinuxLabels(content)

	if !strings.Contains(result, "Volume=/host/path:/container/path:ro,z") {
		t.Errorf("expected :ro,z appended to bind mount, got:\n%s", result)
	}
}

func TestInjectNetworkAliases_SingleAlias(t *testing.T) {
	content := "[Container]\nImage=nginx"
	c := &Cooker{ProjectName: "myproject"}
	result := c.injectNetworkAliases(content, "cq-myproject-web.container")

	if !strings.Contains(result, "NetworkAlias=web") {
		t.Errorf("expected NetworkAlias=web, got:\n%s", result)
	}
}

func TestInjectNetworkAliases_WithContainerName(t *testing.T) {
	content := "[Container]\nImage=nginx\nContainerName=my-custom-name"
	c := &Cooker{ProjectName: "myproject"}
	result := c.injectNetworkAliases(content, "cq-myproject-web.container")

	if !strings.Contains(result, "NetworkAlias=web") {
		t.Errorf("expected NetworkAlias=web, got:\n%s", result)
	}
	if !strings.Contains(result, "NetworkAlias=my-custom-name") {
		t.Errorf("expected NetworkAlias=my-custom-name, got:\n%s", result)
	}
}

func TestInjectNetworkAliases_Idempotent(t *testing.T) {
	content := "[Container]\nImage=nginx\nNetworkAlias=existing"
	c := &Cooker{ProjectName: "myproject"}
	result := c.injectNetworkAliases(content, "cq-myproject-web.container")

	count := strings.Count(result, "NetworkAlias=")
	if count != 1 {
		t.Errorf("expected exactly 1 NetworkAlias (idempotent), got %d:\n%s", count, result)
	}
}

func TestInjectNetworkAliases_NoContainerSection(t *testing.T) {
	content := "[Image]\nName=nginx"
	c := &Cooker{ProjectName: "myproject"}
	result := c.injectNetworkAliases(content, "cq-myproject-web.container")

	if result != content {
		t.Errorf("expected no modification without [Container] section, got:\n%s", result)
	}
}

func TestInjectNetworkAliases_WithExistingLabels(t *testing.T) {
	content := "[Container]\nImage=nginx\nLabel=com.comquad.managed=true"
	c := &Cooker{ProjectName: "myproject"}
	result := c.injectNetworkAliases(content, "cq-myproject-web.container")

	if !strings.Contains(result, "NetworkAlias=web") {
		t.Errorf("expected NetworkAlias=web, got:\n%s", result)
	}
	if !strings.Contains(result, "Label=com.comquad.managed=true") {
		t.Errorf("expected existing labels preserved, got:\n%s", result)
	}
}

func TestLabelFields_SimpleKeyValue(t *testing.T) {
	result := labelFields("app.name=myapp")
	if len(result) != 1 || result[0] != "app.name=myapp" {
		t.Errorf("expected [app.name=myapp], got %v", result)
	}
}

func TestLabelFields_MultiplePairs(t *testing.T) {
	result := labelFields("app.name=myapp app.env=production")
	if len(result) != 2 {
		t.Fatalf("expected 2 fields, got %d: %v", len(result), result)
	}
	if result[0] != "app.name=myapp" {
		t.Errorf("expected app.name=myapp, got %q", result[0])
	}
	if result[1] != "app.env=production" {
		t.Errorf("expected app.env=production, got %q", result[1])
	}
}

func TestLabelFields_QuotedValue(t *testing.T) {
	result := labelFields(`app.owner="Alice Bob"`)
	if len(result) != 1 {
		t.Fatalf("expected 1 field, got %d: %v", len(result), result)
	}
	if result[0] != `app.owner="Alice Bob"` {
		t.Errorf("expected app.owner=\"Alice Bob\", got %q", result[0])
	}
}

func TestLabelFields_SingleQuotedValue(t *testing.T) {
	result := labelFields(`app.owner='Alice Bob'`)
	if len(result) != 1 {
		t.Fatalf("expected 1 field, got %d: %v", len(result), result)
	}
	if result[0] != `app.owner='Alice Bob'` {
		t.Errorf("expected app.owner='Alice Bob', got %q", result[0])
	}
}

func TestLabelFields_MixedQuotedAndUnquoted(t *testing.T) {
	result := labelFields(`app.name=myapp app.owner="Alice Bob" app.env=production`)
	if len(result) != 3 {
		t.Fatalf("expected 3 fields, got %d: %v", len(result), result)
	}
	if result[0] != "app.name=myapp" {
		t.Errorf("expected app.name=myapp, got %q", result[0])
	}
	if result[1] != `app.owner="Alice Bob"` {
		t.Errorf("expected app.owner=\"Alice Bob\", got %q", result[1])
	}
	if result[2] != "app.env=production" {
		t.Errorf("expected app.env=production, got %q", result[2])
	}
}

func TestLabelFields_EscapedQuoteInValue(t *testing.T) {
	result := labelFields(`app.msg="hello \"world\""`)
	if len(result) != 1 {
		t.Fatalf("expected 1 field, got %d: %v", len(result), result)
	}
	if result[0] != `app.msg="hello \"world\""` {
		t.Errorf("expected app.msg=\"hello \\\"world\\\"\", got %q", result[0])
	}
}

func TestLabelFields_EmptyString(t *testing.T) {
	result := labelFields("")
	if len(result) != 0 {
		t.Errorf("expected 0 fields, got %d: %v", len(result), result)
	}
}

func TestLabelFields_WhitespaceOnly(t *testing.T) {
	result := labelFields("   ")
	if len(result) != 0 {
		t.Errorf("expected 0 fields, got %d: %v", len(result), result)
	}
}

func TestLabelFields_LeadingTrailingSpaces(t *testing.T) {
	result := labelFields("  app.name=myapp  ")
	if len(result) != 1 || result[0] != "app.name=myapp" {
		t.Errorf("expected [app.name=myapp], got %v", result)
	}
}

func TestLabelFields_BareKeyWithoutValue(t *testing.T) {
	result := labelFields("barekey")
	if len(result) != 1 || result[0] != "barekey" {
		t.Errorf("expected [barekey], got %v", result)
	}
}

func TestLabelFields_EmptyValue(t *testing.T) {
	result := labelFields("app.name=")
	if len(result) != 1 || result[0] != "app.name=" {
		t.Errorf("expected [app.name=], got %v", result)
	}
}

func TestLabelFields_UnclosedQuote(t *testing.T) {
	result := labelFields(`app.owner="Alice Bob`)
	if len(result) != 1 || result[0] != `app.owner="Alice Bob` {
		t.Errorf("expected [app.owner=\"Alice Bob], got %v", result)
	}
}

func TestLabelFields_MultipleSpacesBetweenPairs(t *testing.T) {
	result := labelFields("app.name=myapp    app.env=production")
	if len(result) != 2 {
		t.Fatalf("expected 2 fields, got %d: %v", len(result), result)
	}
	if result[0] != "app.name=myapp" {
		t.Errorf("expected app.name=myapp, got %q", result[0])
	}
	if result[1] != "app.env=production" {
		t.Errorf("expected app.env=production, got %q", result[1])
	}
}
