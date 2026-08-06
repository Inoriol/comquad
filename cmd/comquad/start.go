package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var startCmd = &cobra.Command{
	Use:   "start [service ...]",
	Short: "Start all units for the project or specific services",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Start(args, dryRun)
	},
}

func init() {
	startCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	startCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show which units would be started without actually starting them")
}
