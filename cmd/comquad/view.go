package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var viewCmd = &cobra.Command{
	Use:     "view [project] [service]",
	Aliases: []string{"overview"},
	Short:   "View systemd units for a project or display a specific unit file",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}

		var projectArg string
		if len(args) > 0 {
			projectArg = args[0]
		}

		return o.View(projectArg)
	},
}

func init() {
	viewCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
}
