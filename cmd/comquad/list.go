package main

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/output"
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

		if output.IsJSONMode() {
			jsonProjects := make([]output.ProjectJSON, 0)
			for _, p := range projects {
				if projectName != "" && p.ProjectName != projectName {
					continue
				}
				jp := output.ProjectJSON{
					Name:       p.ProjectName,
					SourcePath: p.SourcePath,
					Files:      len(p.Files),
				}
				if p.Resources != nil {
					jp.Resources = &output.ResourcesJSON{
						Containers: p.Resources.Containers,
						Networks:   p.Resources.Networks,
						Volumes:    p.Resources.Volumes,
						Images:     p.Resources.Images,
						Builds:     p.Resources.Builds,
					}
				}
				jsonProjects = append(jsonProjects, jp)
			}
			return output.PrintJSON(&output.ProjectListData{Projects: jsonProjects})
		}

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
