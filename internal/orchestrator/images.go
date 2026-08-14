package orchestrator

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/reconcile"
)

func normalizeImageRef(ref string) string {
	if ref == "" || strings.Contains(ref, "${") {
		return ref
	}
	if !strings.Contains(ref, "/") {
		return "docker.io/library/" + ref
	}
	parts := strings.SplitN(ref, "/", 2)
	first := parts[0]
	if first == "docker.io" {
		return ref
	}
	if strings.Contains(first, ".") || isRegistryWithPort(first) || first == "localhost" {
		return ref
	}
	return "docker.io/" + ref
}

func isRegistryWithPort(s string) bool {
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return false
	}
	port := s[idx+1:]
	if len(port) == 0 || len(port) > 5 {
		return false
	}
	for _, c := range port {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

type PullStrategy string

const (
	PullAlways  PullStrategy = "always"
	PullMissing PullStrategy = "missing"
	PullNever   PullStrategy = "never"
)

func ParsePullStrategy(s string) (PullStrategy, error) {
	switch strings.ToLower(s) {
	case "always":
		return PullAlways, nil
	case "missing":
		return PullMissing, nil
	case "never":
		return PullNever, nil
	default:
		return "", fmt.Errorf("invalid pull strategy: %s (must be 'always', 'missing', or 'never')", s)
	}
}

func imageExists(image string) bool {
	cmd := exec.Command("podman", "image", "inspect", image)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() != 125 {
			logger.Warn(fmt.Sprintf("podman image inspect failed for %s: %v", image, err))
		}
		return false
	}
	return true
}

func pullImage(image string) error {
	logger.Action("Pulling image: " + image)
	cmd := exec.Command("podman", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}
	return nil
}

func handleImage(strategy PullStrategy, image string) error {
	switch strategy {
	case PullAlways:
		return pullImage(image)
	case PullNever:
		if !imageExists(image) {
			return fmt.Errorf("image %s not found locally and pull strategy is 'never'", image)
		}
		logger.Action("Using local image: " + image)
		return nil
	case PullMissing:
		if imageExists(image) {
			logger.Action("Image already exists locally: " + image)
			return nil
		}
		return pullImage(image)
	default:
		return fmt.Errorf("unknown pull strategy: %s", strategy)
	}
}

func (o *Orchestrator) handleImages(projectFiles []string, units []c2q.QuadletUnit, pullStrategy string) error {
	strat, err := ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	for _, unit := range units {
		if unit.Type != c2q.UnitContainer {
			continue
		}

		if hasBuildUnit(units, unit.Name) {
			logger.Action("Skipping pull for " + unit.Name + " (built from .build quadlet)")
			continue
		}

		image := getDirective(unit, c2q.SectionContainer, "Image")
		if image == "" {
			continue
		}

		image = resolveImageRef(units, image)

		if err := handleImage(strat, image); err != nil {
			return fmt.Errorf("failed to handle image %s: %w", image, err)
		}
		logger.Success("Handled image: " + image)
	}

	return nil
}

func getDirective(unit c2q.QuadletUnit, sectionName, key string) string {
	for _, sec := range unit.Sections {
		if sec.Name == sectionName {
			for _, d := range sec.Directives {
				if d.Key == key && len(d.Values) > 0 {
					return d.Values[0]
				}
			}
		}
	}
	return ""
}

func resolveImageRef(units []c2q.QuadletUnit, ref string) string {
	if !strings.HasSuffix(ref, ".image") {
		return ref
	}
	imageUnitName := strings.TrimSuffix(ref, ".image")
	for _, unit := range units {
		if unit.Name == imageUnitName && unit.Type == c2q.UnitImage {
			if img := getDirective(unit, c2q.SectionImage, "Image"); img != "" {
				return img
			}
		}
	}
	return ref
}

func hasBuildUnit(units []c2q.QuadletUnit, containerName string) bool {
	for _, unit := range units {
		if unit.Type == c2q.UnitBuild && unit.Name == containerName {
			return true
		}
	}
	return false
}

func hasBuildFile(projectFiles []string, containerPath string) bool {
	containerBase := filepath.Base(containerPath)
	buildBase := strings.TrimSuffix(containerBase, ".container") + ".build"
	for _, f := range projectFiles {
		if filepath.Base(f) == buildBase {
			return true
		}
	}
	return false
}

func (o *Orchestrator) printDryRun(units []c2q.QuadletUnit, targetDir string, pullStrategy string, plan reconcile.Plan) error {
	logger.Printf("Dry run — project: %s\n", o.projectName)
	logger.Printf("Target directory: %s\n\n", targetDir)

	strat, err := ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	for _, unit := range units {
		if unit.Type != c2q.UnitContainer {
			continue
		}

		image := getDirective(unit, c2q.SectionContainer, "Image")
		if image == "" {
			continue
		}

		image = resolveImageRef(units, image)

		if hasBuildUnit(units, unit.Name) {
			logger.Printf("[build] %-12s %s  (would be built locally, no pull)\n", unit.Name+".container", image)
			continue
		}

		switch strat {
		case PullAlways:
			logger.Printf("[image] %-12s %s  (would pull: always)\n", unit.Name+".container", image)
		case PullMissing:
			if imageExists(image) {
				logger.Printf("[image] %-12s %s  (already exists locally, would skip pull)\n", unit.Name+".container", image)
			} else {
				logger.Printf("[image] %-12s %s  (would pull: not found locally)\n", unit.Name+".container", image)
			}
		case PullNever:
			logger.Printf("[image] %-12s %s  (pull skipped: never)\n", unit.Name+".container", image)
		}
	}

	var created, changed, removed []reconcile.FilePlan
	for _, fp := range plan.Files {
		switch fp.Status {
		case reconcile.StatusCreated:
			created = append(created, fp)
		case reconcile.StatusChanged:
			changed = append(changed, fp)
		case reconcile.StatusRemoved:
			removed = append(removed, fp)
		}
	}

	if len(created)+len(changed)+len(removed) == 0 {
		logger.Print("\nNo changes — quadlet files are up to date.\n")
	} else {
		logger.Printf("\n%d file(s) to write, %d to change, %d to remove:\n\n", len(created), len(changed), len(removed))
	}

	separator := strings.Repeat("─", 60)

	for _, fp := range created {
		logger.Print(separator)
		logger.Printf("  %s  (new)\n", fp.TargetPath)
		logger.Print(separator)
		logger.Print(strings.TrimRight(fp.NewContent, "\n"))
		logger.Print("")
	}
	for _, fp := range changed {
		logger.Print(separator)
		logger.Printf("  %s  (changed)\n", fp.TargetPath)
		logger.Print(separator)
		fmt.Print(colorizeDiff(fp.Diff()))
		logger.Print("")
	}
	for _, fp := range removed {
		logger.Print(separator)
		logger.Printf("  %s  (removed)\n", fp.TargetPath)
		logger.Print(separator)
		fmt.Print(colorizeDiff(fp.Diff()))
		logger.Print("")
	}

	for _, unit := range units {
		if unit.Type == c2q.UnitBuild {
			file := getDirective(unit, c2q.SectionBuild, "File")
			if file != "" && filepath.IsAbs(file) {
				logger.Printf("  Dockerfile → %s\n", file)
				if wd := getDirective(unit, c2q.SectionBuild, "SetWorkingDirectory"); wd != "" {
					src := filepath.Join(wd, "Dockerfile")
					if _, err := os.Stat(src); os.IsNotExist(err) {
						src = filepath.Join(wd, "Containerfile")
					}
					if patched, err := c2q.PatchDockerfileFile(src); err == nil {
						logger.Print("\n── patched ────────────────────────────────────────────────────────────")
						logger.Print(strings.TrimRight(patched, "\n"))
						logger.Print("── end of patch ───────────────────────────────────────────────────────\n")
					}
				}
			}
		}
	}

	logger.Print("Dry run complete — nothing was written, no units started.")
	return nil
}

func stripServiceName(units []c2q.QuadletUnit) {
	for i := range units {
		if units[i].Type != c2q.UnitContainer {
			continue
		}
		for j := range units[i].Sections {
			if units[i].Sections[j].Name != c2q.SectionContainer {
				continue
			}
			dirs := units[i].Sections[j].Directives
			filtered := dirs[:0]
			for _, d := range dirs {
				if d.Key != "ServiceName" {
					filtered = append(filtered, d)
				}
			}
			units[i].Sections[j].Directives = filtered
		}
	}
}

func resolveContainerImages(units []c2q.QuadletUnit) {
	for i := range units {
		if units[i].Type != c2q.UnitContainer {
			continue
		}
		for j := range units[i].Sections {
			sec := &units[i].Sections[j]
			if sec.Name != c2q.SectionContainer {
				continue
			}
			for k, d := range sec.Directives {
				if d.Key != "Image" || len(d.Values) == 0 {
					continue
				}
				resolved := resolveImageRef(units, d.Values[0])
				resolved = normalizeImageRef(resolved)
				sec.Directives[k].Values[0] = resolved
			}
		}
	}
}
