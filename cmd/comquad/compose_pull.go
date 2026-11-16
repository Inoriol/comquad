package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var composePullCmd = &cobra.Command{
	Use:   "pull [service...]",
	Short: "Pull service images",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(resolveComposeProjectName())
		if err != nil {
			return err
		}
		if composeFile != "" {
			o.WithComposeFile(composeFile)
		}

		return o.Pull(orchestrator.PullOptions{
			PullStrategy: "always",
			DryRun:       dryRun,
			Services:     args,
		})
	},
}
