package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var pullCmd = &cobra.Command{
	Use:   "pull [service...]",
	Short: "Pull images defined in compose.yaml without starting containers",
	Example: `  comquad pull
  comquad pull web db                        # Pull images for specific services
  comquad pull --dry-run                     # Preview without pulling
  comquad pull -n my-project                 # Override project name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}

		return o.Pull(orchestrator.PullOptions{
			PullStrategy: "always",
			DryRun:       dryRun,
			Services:     args,
		})
	},
}

func init() {
	pullCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	pullCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without pulling anything")
}
