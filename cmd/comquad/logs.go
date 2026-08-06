package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var follow bool
var logTail string
var logSince string
var logTime bool

var logsCmd = &cobra.Command{
	Use:   "logs [service ...]",
	Short: "Print logs for a deployed project",
	Example: `  comquad logs                 # All services (one-shot)
  comquad logs -f              # Follow all service logs
  comquad logs web             # Single service
  comquad logs --tail 50       # Last 50 lines
  comquad logs --since 10m     # Last 10 minutes
  comquad logs --since "2024-01-01 12:00:00"
  comquad logs -t              # Include RFC3339Nano timestamps`,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		follow, _ := cmd.Flags().GetBool("follow")
		return o.Logs(args, follow, logTail, logSince, logTime)
	},
}

func init() {
	logsCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().StringVar(&logTail, "tail", "", "Number of lines to show from the end of logs (default: all)")
	logsCmd.Flags().StringVar(&logSince, "since", "", "Show logs since a specific time (e.g. '10m', '2024-01-01 12:00:00')")
	logsCmd.Flags().BoolVarP(&logTime, "time", "t", false, "Display timestamps in RFC3339Nano format")
}
