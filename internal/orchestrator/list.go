package orchestrator

import (
	"fmt"
	"strings"

	"github.com/Inoriol/comquad/internal/logger"
)

// List prints all currently deployed projects and their files.
// When filter is non-empty, only the matching project is shown.
func (o *Orchestrator) List(filter string) error {
	stateMgr, err := o.newState()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	projects := stateMgr.ListProjects()

	if len(projects) == 0 {
		logger.Print("No projects currently deployed.")
		return nil
	}

	logger.Printf("%-20s %-40s %s\n", "PROJECT", "SOURCE", "FILES")
	logger.Print(strings.Repeat("-", 72))
	for _, p := range projects {
		if filter != "" && p.ProjectName != filter {
			continue
		}
		logger.Printf("%-20s %-40s %d units\n", p.ProjectName, p.SourcePath, len(p.Files))
	}

	return nil
}
