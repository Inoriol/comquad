package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"comquad/internal/orchestrator"
)

var force bool

var regenerateCmd = &cobra.Command{
	Use:   "regenerate",
	Short: "Restore state from Podman labels",
	Example: `  comquad regenerate --force
  comquad regenerate --force --dry-run     # Preview without writing`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !force {
			return fmt.Errorf("regenerate requires --force to overwrite existing state")
		}
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Regenerate(dryRun)
	},
}

func init() {
	regenerateCmd.Flags().BoolVar(&force, "force", false, "Force regeneration of state file")
	regenerateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be regenerated without writing")
}
