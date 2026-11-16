package deploy

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const AutoNSRangeSize = 65535

type SubuidEntry struct {
	Name  string
	Start int
	Size  int
}

func ParseSubuid() ([]SubuidEntry, error) {
	return ParseSubuidFile("/etc/subuid")
}

func ParseSubuidFile(path string) ([]SubuidEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []SubuidEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 3 {
			continue
		}
		start, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		size, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		entries = append(entries, SubuidEntry{
			Name:  parts[0],
			Start: start,
			Size:  size,
		})
	}
	return entries, scanner.Err()
}

func FindHighestAllocatedEnd(entries []SubuidEntry) int {
	highest := 0
	for _, e := range entries {
		end := e.Start + e.Size
		if end > highest {
			highest = end
		}
	}
	return highest
}

func RegistryStart(entries []SubuidEntry) int {
	highest := FindHighestAllocatedEnd(entries)
	if highest == 0 {
		return AutoNSRangeSize
	}
	return ((highest + AutoNSRangeSize - 1) / AutoNSRangeSize) * AutoNSRangeSize
}

func AllocateStackRange(entries []SubuidEntry, usedBases []int) int {
	regStart := RegistryStart(entries)

	used := make(map[int]bool, len(usedBases))
	for _, b := range usedBases {
		used[b] = true
	}

	for candidate := regStart; ; candidate += AutoNSRangeSize {
		if !used[candidate] {
			return candidate
		}
	}
}

func ValidateContainersEntry() error {
	return ValidateContainersEntryFile("/etc/subuid")
}

func ValidateContainersEntryFile(path string) error {
	entries, err := ParseSubuidFile(path)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	for _, e := range entries {
		if e.Name == "containers" {
			return nil
		}
	}
	return fmt.Errorf("/etc/subuid missing 'containers' entry required for AutoNS")
}

func FindContainersEntry(entries []SubuidEntry) *SubuidEntry {
	for i, e := range entries {
		if e.Name == "containers" {
			return &entries[i]
		}
	}
	return nil
}

func CollectUsedBases(projects []ProjectState) []int {
	var bases []int
	for _, p := range projects {
		if p.AutoNSRange != nil {
			bases = append(bases, p.AutoNSRange.BaseUID)
		}
	}
	sort.Ints(bases)
	return bases
}
