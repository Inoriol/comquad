package main

import (
	"github.com/spf13/cobra"
)

var composeFile string
var composeProjectName string

var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Docker Compose compatible commands",
	Long: `Provides docker-compose compatible CLI syntax while using comquad's systemd-native workflow.

This allows users to use familiar docker-compose commands while getting comquad's unique
features like diffs, confirmations, and quadlet generation.

Key differences from docker-compose:
  - 'up' without flags follows logs (like 'docker compose up'), use '-d' to detach
  - Changes show a diff and ask for confirmation (use '--no-diff' to skip)
  - Service manager is systemd (via quadlet), not direct container ops`,
}

func init() {
	composeCmd.PersistentFlags().StringVarP(&composeFile, "file", "f", "", "Compose file path (file or directory)")
	composeCmd.PersistentFlags().StringVarP(&composeProjectName, "project-name", "p", "", "Project name")
	composeCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview without applying changes")

	composeCmd.AddCommand(composeUpCmd)
	composeCmd.AddCommand(composeDownCmd)
	composeCmd.AddCommand(composePsCmd)
	composeCmd.AddCommand(composeLogsCmd)
	composeCmd.AddCommand(composeStartCmd)
	composeCmd.AddCommand(composeStopCmd)
	composeCmd.AddCommand(composeRestartCmd)
	composeCmd.AddCommand(composeExecCmd)
	composeCmd.AddCommand(composeBuildCmd)
	composeCmd.AddCommand(composePullCmd)
}
