package orchestrator

import (
	"fmt"
	"os"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
)

// Regenerate scans Podman for managed resources and reconstructs the state file.
// When dryRun is true, it prints what would be written without saving the state file.
func (o *Orchestrator) Regenerate(dryRun bool) error {
	stateMgr, err := deploy.RegenerateState()
	if err != nil {
		return err
	}

	projects := stateMgr.ListProjects()
	if len(projects) == 0 {
		logger.Print("No managed projects found in Podman.")
		return nil
	}

	if dryRun {
		logger.Printf("Dry run — would regenerate %d project(s) from Podman labels:\n\n", len(projects))
	} else {
		logger.Printf("Discovered %d project(s) from Podman labels:\n\n", len(projects))
	}

	for _, p := range projects {
		resources := p.Resources
		if resources == nil {
			resources = &deploy.ResourceInfo{}
		}

		total := len(resources.Containers) + len(resources.Networks) + len(resources.Volumes) + len(resources.Images) + len(resources.Builds)
		logger.Printf("  %s (%d resource%s)\n", p.ProjectName, total, pluralize(total))

		for _, c := range resources.Containers {
			logger.Printf("    Container:  %s\n", c)
		}
		for _, n := range resources.Networks {
			logger.Printf("    Network:    %s\n", n)
		}
		for _, v := range resources.Volumes {
			logger.Printf("    Volume:     %s\n", v)
		}

		if len(p.Files) > 0 {
			logger.Printf("    Quadlet files: %d\n", len(p.Files))
		} else {
			logger.Printf("    Quadlet files: 0 (not found in systemd directory)\n")
		}
		logger.Print("")
	}

	if dryRun {
		logger.Printf("Dry run — state file not written: %s\n", stateMgr.StateFilePath)
		return nil
	}

	if err := stateMgr.Save(); err != nil {
		return fmt.Errorf("failed to save state file: %w", err)
	}

	// Baseline integrity is unknown after a rebuild, so clear it to force a
	// clean 2-way reconcile on the next `up`.
	for _, p := range projects {
		if dir, err := resolveBaselineDir(p.ProjectName); err == nil {
			os.RemoveAll(dir)
		}
	}

	logger.Printf("Regenerated state file: %s\n", stateMgr.StateFilePath)
	return nil
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
