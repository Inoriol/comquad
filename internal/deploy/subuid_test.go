package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSubuidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subuid")

	content := `root:100000:65536
containers:2147483647:2147483648
inoriol:1000000:65536
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ParseSubuidFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Name != "root" || entries[0].Start != 100000 || entries[0].Size != 65536 {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Name != "containers" || entries[1].Start != 2147483647 || entries[1].Size != 2147483648 {
		t.Errorf("unexpected containers entry: %+v", entries[1])
	}
}

func TestParseSubuidFile_Comments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subuid")

	content := `# This is a comment
root:100000:65536

# Another comment
containers:2147483647:2147483648
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ParseSubuidFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (comments skipped), got %d", len(entries))
	}
}

func TestFindHighestAllocatedEnd(t *testing.T) {
	entries := []SubuidEntry{
		{Name: "root", Start: 100000, Size: 65536},
		{Name: "containers", Start: 2147483647, Size: 2147483648},
	}

	highest := FindHighestAllocatedEnd(entries)
	expected := 2147483647 + 2147483648
	if highest != expected {
		t.Errorf("expected %d, got %d", expected, highest)
	}
}

func TestRegistryStart(t *testing.T) {
	tests := []struct {
		name     string
		entries  []SubuidEntry
		expected int
	}{
		{
			name:     "empty entries",
			entries:  nil,
			expected: AutoNSRangeSize,
		},
		{
			name: "aligned end",
			entries: []SubuidEntry{
				{Name: "root", Start: 0, Size: 65535},
			},
			expected: 65535,
		},
		{
			name: "unaligned end",
			entries: []SubuidEntry{
				{Name: "root", Start: 0, Size: 100000},
			},
			expected: 131070,
		},
		{
			name: "containers entry at high range",
			entries: []SubuidEntry{
				{Name: "containers", Start: 2147483647, Size: 2147483648},
			},
			expected: 4294967295,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RegistryStart(tt.entries)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestAllocateStackRange(t *testing.T) {
	entries := []SubuidEntry{
		{Name: "root", Start: 100000, Size: 65536},
	}

	regStart := RegistryStart(entries)

	t.Run("first allocation", func(t *testing.T) {
		base := AllocateStackRange(entries, nil)
		if base != regStart {
			t.Errorf("expected %d, got %d", regStart, base)
		}
	})

	t.Run("second allocation skips used", func(t *testing.T) {
		usedBases := []int{regStart}
		base := AllocateStackRange(entries, usedBases)
		expected := regStart + AutoNSRangeSize
		if base != expected {
			t.Errorf("expected %d, got %d", expected, base)
		}
	})

	t.Run("skips gaps", func(t *testing.T) {
		usedBases := []int{regStart, regStart + AutoNSRangeSize*2}
		base := AllocateStackRange(entries, usedBases)
		expected := regStart + AutoNSRangeSize
		if base != expected {
			t.Errorf("expected %d (first gap), got %d", expected, base)
		}
	})
}

func TestValidateContainersEntryFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		err := ValidateContainersEntryFile(filepath.Join(dir, "nonexistent"))
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("no containers entry", func(t *testing.T) {
		path := filepath.Join(dir, "subuid_no_containers")
		content := `root:100000:65536
inoriol:1000000:65536
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		err := ValidateContainersEntryFile(path)
		if err == nil {
			t.Error("expected error for missing containers entry")
		}
	})

	t.Run("has containers entry", func(t *testing.T) {
		path := filepath.Join(dir, "subuid_with_containers")
		content := `root:100000:65536
containers:2147483647:2147483648
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		err := ValidateContainersEntryFile(path)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestFindContainersEntry(t *testing.T) {
	entries := []SubuidEntry{
		{Name: "root", Start: 100000, Size: 65536},
		{Name: "containers", Start: 2147483647, Size: 2147483648},
	}

	entry := FindContainersEntry(entries)
	if entry == nil {
		t.Fatal("expected to find containers entry")
	}
	if entry.Name != "containers" {
		t.Errorf("expected name 'containers', got %q", entry.Name)
	}
	if entry.Start != 2147483647 {
		t.Errorf("expected start 2147483647, got %d", entry.Start)
	}
}

func TestFindContainersEntry_NotFound(t *testing.T) {
	entries := []SubuidEntry{
		{Name: "root", Start: 100000, Size: 65536},
	}

	entry := FindContainersEntry(entries)
	if entry != nil {
		t.Errorf("expected nil, got %+v", entry)
	}
}

func TestCollectUsedBases(t *testing.T) {
	projects := []ProjectState{
		{ProjectName: "app1", AutoNSRange: &AutoNSRange{BaseUID: 65535}},
		{ProjectName: "app2"},
		{ProjectName: "app3", AutoNSRange: &AutoNSRange{BaseUID: 196605}},
	}

	bases := CollectUsedBases(projects)
	if len(bases) != 2 {
		t.Fatalf("expected 2 bases, got %d", len(bases))
	}
	if bases[0] != 65535 || bases[1] != 196605 {
		t.Errorf("expected [65535, 196605], got %v", bases)
	}
}
