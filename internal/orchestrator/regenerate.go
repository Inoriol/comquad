package orchestrator

import (
	"fmt"

	"comquad/internal/deploy"
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
		fmt.Println("No managed projects found in Podman.")
		return nil
	}

	if dryRun {
		fmt.Printf("Dry run — would regenerate %d project(s) from Podman labels:\n\n", len(projects))
	} else {
		fmt.Printf("Discovered %d project(s) from Podman labels:\n\n", len(projects))
	}

	for _, p := range projects {
		resources := p.Resources
		if resources == nil {
			resources = &deploy.ResourceInfo{}
		}

		total := len(resources.Containers) + len(resources.Networks) + len(resources.Volumes)
		fmt.Printf("  %s (%d resource%s)\n", p.ProjectName, total, pluralize(total))

		for _, c := range resources.Containers {
			fmt.Printf("    Container:  %s\n", c)
		}
		for _, n := range resources.Networks {
			fmt.Printf("    Network:    %s\n", n)
		}
		for _, v := range resources.Volumes {
			fmt.Printf("    Volume:     %s\n", v)
		}

		if len(p.Files) > 0 {
			fmt.Printf("    Quadlet files: %d\n", len(p.Files))
		} else {
			fmt.Printf("    Quadlet files: 0 (not found in systemd directory)\n")
		}
		fmt.Println()
	}

	if dryRun {
		fmt.Printf("Dry run — state file not written: %s\n", stateMgr.StateFilePath)
		return nil
	}

	if err := stateMgr.Save(); err != nil {
		return fmt.Errorf("failed to save state file: %w", err)
	}

	fmt.Printf("Regenerated state file: %s\n", stateMgr.StateFilePath)
	return nil
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
