package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var composeLogsFollow bool
var composeLogsTail string
var composeLogsSince string
var composeLogsTimestamps bool

var composeLogsCmd = &cobra.Command{
	Use:   "logs [service...]",
	Short: "View output from containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(resolveComposeProjectName())
		if err != nil {
			return err
		}
		if composeFile != "" {
			o.WithComposeFile(composeFile)
		}
		return o.Logs(args, composeLogsFollow, composeLogsTail, composeLogsSince, composeLogsTimestamps)
	},
}

func init() {
	composeLogsCmd.Flags().BoolVarP(&composeLogsFollow, "follow", "f", false, "Follow log output")
	composeLogsCmd.Flags().StringVar(&composeLogsTail, "tail", "", "Number of lines to show from the end of logs")
	composeLogsCmd.Flags().StringVar(&composeLogsSince, "since", "", "Show logs since a specific time")
	composeLogsCmd.Flags().BoolVarP(&composeLogsTimestamps, "timestamps", "t", false, "Show timestamps")
}
