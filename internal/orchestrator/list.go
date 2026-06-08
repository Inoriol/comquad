package orchestrator

import (
	"fmt"
	"strings"

	"comquad/internal/deploy"
)

// List prints all currently deployed projects and their files
func (o *Orchestrator) List() error {
	stateMgr, err := deploy.NewStateManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	projects := stateMgr.ListProjects()

	if len(projects) == 0 {
		fmt.Println("No projects currently deployed.")
		return nil
	}

	fmt.Printf("%-20s %-40s %s\n", "PROJECT", "SOURCE", "FILES")
	fmt.Println(strings.Repeat("-", 72))
	for _, p := range projects {
		if o.projectName != "" && p.ProjectName != o.projectName {
			continue
		}
		fmt.Printf("%-20s %-40s %d units\n", p.ProjectName, p.SourcePath, len(p.Files))
	}

	return nil
}
