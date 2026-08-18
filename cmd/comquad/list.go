package main

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all currently deployed projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		stateMgr, err := deploy.NewStateManager()
		if err != nil {
			return err
		}

		projects := stateMgr.ListProjects()

		if len(projects) == 0 {
			logger.Print("No projects currently deployed.")
			return nil
		}

		logger.Printf("%-20s %-40s %s\n", "PROJECT", "SOURCE", "FILES")
		logger.Print(strings.Repeat("-", 72))
		for _, p := range projects {
			if projectName != "" && p.ProjectName != projectName {
				continue
			}
			logger.Printf("%-20s %-40s %d units\n", p.ProjectName, p.SourcePath, len(p.Files))
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringVarP(&projectName, "name", "n", "", "Filter by project name")
}
