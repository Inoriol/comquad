package orchestrator

import (
	"fmt"
	"path/filepath"
	"strings"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/output"
	"github.com/Inoriol/comquad/internal/reconcile"
)

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

func setPolicyOnImageUnits(units []c2q.QuadletUnit, strategy PullStrategy) {
	for i := range units {
		if units[i].Type != c2q.UnitImage {
			continue
		}
		for j := range units[i].Sections {
			if units[i].Sections[j].Name != c2q.SectionImage {
				continue
			}
			dirs := units[i].Sections[j].Directives
			found := false
			for k := range dirs {
				if dirs[k].Key == "Policy" {
					dirs[k].Values = []string{string(strategy)}
					found = true
					break
				}
			}
			if !found {
				units[i].Sections[j].Directives = append(dirs, c2q.Directive{
					Key:    "Policy",
					Values: []string{string(strategy)},
				})
			}
		}
	}
}

func (o *Orchestrator) handleImages(projectFiles []string, units []c2q.QuadletUnit, pullStrategy string) error {
	strat, err := ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".image") {
			continue
		}

		unitName := ImageFileToUnitName(f)
		baseName := strings.TrimSuffix(filepath.Base(f), ".image")

		if hasBuildUnitForName(units, baseName) {
			if !output.IsJSONMode() {
				logger.Info("Skipping image unit " + unitName + " (has corresponding .build unit)")
			}
			continue
		}

		if err := dbusMgr.WaitForUnit(unitName, startUnitWaitTime); err != nil {
			logger.Warn(fmt.Sprintf("image unit %s not produced by quadlet generator, skipping: %v", unitName, err))
			continue
		}

		if !output.IsJSONMode() {
			logger.Action("Handling image unit: " + unitName)
		}

		switch strat {
		case PullAlways:
			if !output.IsJSONMode() {
				logger.Action("Stopping image unit for re-pull: " + unitName)
			}
			if err := dbusMgr.StopUnit(unitName); err != nil {
				logger.Warn(fmt.Sprintf("failed to stop image unit %s: %v", unitName, err))
			}
			if err := dbusMgr.StartUnit(unitName); err != nil {
				return fmt.Errorf("failed to start image unit %s: %w", unitName, err)
			}
		case PullMissing:
			statuses, err := dbusMgr.ListUnitsByNames([]string{unitName})
			if err != nil || len(statuses) == 0 {
				if err := dbusMgr.StartUnit(unitName); err != nil {
					return fmt.Errorf("failed to start image unit %s: %w", unitName, err)
				}
			} else {
				status := statuses[0]
				if status.SubState != "exited" {
					if err := dbusMgr.StartUnit(unitName); err != nil {
						return fmt.Errorf("failed to start image unit %s: %w", unitName, err)
					}
				} else {
					if !output.IsJSONMode() {
						logger.Info("Image unit already pulled: " + unitName)
					}
				}
			}
		case PullNever:
			if err := dbusMgr.StartUnit(unitName); err != nil {
				return fmt.Errorf("failed to verify image unit %s: %w", unitName, err)
			}
		}

		logger.Success("Handled image unit: " + unitName)
	}

	return nil
}

func hasBuildUnitForName(units []c2q.QuadletUnit, name string) bool {
	for _, unit := range units {
		if unit.Type == c2q.UnitBuild && unit.Name == name {
			return true
		}
	}
	return false
}

func hasImageUnitForName(units []c2q.QuadletUnit, name string) bool {
	for _, unit := range units {
		if unit.Type == c2q.UnitImage && unit.Name == name {
			return true
		}
	}
	return false
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

func (o *Orchestrator) printDryRun(units []c2q.QuadletUnit, targetDir string, pullStrategy string, plan reconcile.Plan) error {
	strat, err := ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	if output.IsJSONMode() {
		return o.printDryRunJSON(units, targetDir, strat, plan)
	}

	logger.Printf("Dry run — project: %s\n", o.projectName)
	logger.Printf("Target directory: %s\n\n", targetDir)

	for _, unit := range units {
		if unit.Type != c2q.UnitImage {
			continue
		}

		image := getDirective(unit, c2q.SectionImage, "Image")
		if image == "" {
			continue
		}

		if hasBuildUnitForName(units, unit.Name) {
			logger.Printf("[build] %-12s %s  (would be built locally, no pull)\n", unit.Name+".image", image)
			continue
		}

		switch strat {
		case PullAlways:
			logger.Printf("[image] %-12s %s  (would re-pull: always)\n", unit.Name+".image", image)
		case PullMissing:
			logger.Printf("[image] %-12s %s  (would pull if not already pulled)\n", unit.Name+".image", image)
		case PullNever:
			logger.Printf("[image] %-12s %s  (would verify local image)\n", unit.Name+".image", image)
		}
	}

	for _, unit := range units {
		if unit.Type != c2q.UnitBuild {
			continue
		}

		if hasImageUnitForName(units, unit.Name) {
			continue
		}

		imageTag := getDirective(unit, c2q.SectionBuild, "ImageTag")
		if imageTag == "" {
			imageTag = "unknown"
		}
		logger.Printf("[build] %-12s %s  (would be built locally)\n", unit.Name+".build", imageTag)
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

	logger.Print("Dry run complete — nothing was written, no units started.")
	return nil
}

func (o *Orchestrator) printDryRunJSON(units []c2q.QuadletUnit, targetDir string, strat PullStrategy, plan reconcile.Plan) error {
	data := output.DryRunData{
		Project:      o.projectName,
		TargetDir:    targetDir,
		PullStrategy: string(strat),
	}

	for _, unit := range units {
		if unit.Type != c2q.UnitImage {
			continue
		}
		image := getDirective(unit, c2q.SectionImage, "Image")
		if image == "" {
			continue
		}
		action := "would verify local image"
		if hasBuildUnitForName(units, unit.Name) {
			action = "would be built locally, no pull"
		} else {
			switch strat {
			case PullAlways:
				action = "would re-pull: always"
			case PullMissing:
				action = "would pull if not already pulled"
			case PullNever:
				action = "would verify local image"
			}
		}
		data.Images = append(data.Images, output.DryRunImage{
			Name:   unit.Name + ".image",
			Ref:    image,
			Action: action,
		})
	}

	for _, unit := range units {
		if unit.Type != c2q.UnitBuild {
			continue
		}
		if hasImageUnitForName(units, unit.Name) {
			continue
		}
		imageTag := getDirective(unit, c2q.SectionBuild, "ImageTag")
		if imageTag == "" {
			imageTag = "unknown"
		}
		data.Builds = append(data.Builds, output.DryRunBuild{
			Name:     unit.Name + ".build",
			ImageTag: imageTag,
		})
	}

	for _, fp := range plan.Files {
		if fp.Status == reconcile.StatusUnchanged {
			continue
		}
		var status string
		switch fp.Status {
		case reconcile.StatusCreated:
			status = "created"
		case reconcile.StatusChanged:
			status = "changed"
		case reconcile.StatusRemoved:
			status = "removed"
		}
		df := output.DryRunFile{
			Name:   fp.Name,
			Path:   fp.TargetPath,
			Status: status,
			Diff:   fp.Diff(),
		}
		if fp.Status == reconcile.StatusCreated {
			df.NewContent = fp.NewContent
		}
		data.Files = append(data.Files, df)
	}

	data.HasChanges = len(data.Files) > 0

	return output.PrintJSON(data)
}
