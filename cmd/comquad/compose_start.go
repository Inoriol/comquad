package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var composeStartCmd = &cobra.Command{
	Use:   "start [service...]",
	Short: "Start services",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(resolveComposeProjectName())
		if err != nil {
			return err
		}
		if composeFile != "" {
			o.WithComposeFile(composeFile)
		}
		return o.Start(args, dryRun)
	},
}
