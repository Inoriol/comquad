package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var restartCmd = &cobra.Command{
	Use:   "restart [service ...]",
	Short: "Restart all units for the project or specific services",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Restart(args, dryRun)
	},
}

func init() {
	restartCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	restartCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show which units would be restarted without actually restarting them")
}
