package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var stopCmd = &cobra.Command{
	Use:   "stop [service ...]",
	Short: "Stop all units for the project or specific services",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Stop(args, dryRun)
	},
}

func init() {
	stopCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	stopCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show which units would be stopped without actually stopping them")
}
