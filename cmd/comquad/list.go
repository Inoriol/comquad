package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var listCmd = &cobra.Command{
	Use:     "list [project]",
	Aliases: []string{"ls"},
	Short:   "List all currently deployed projects",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			projectName = args[0]
			o, err := orchestrator.NewOrchestrator(projectName)
			if err != nil {
				return err
			}
			return o.View("")
		}

		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.List(projectName)
	},
}

func init() {
	listCmd.Flags().StringVarP(&projectName, "name", "n", "", "Filter by project name")
}
