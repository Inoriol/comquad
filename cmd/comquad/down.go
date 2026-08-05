package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"comquad/internal/orchestrator"
)

var downRemoveVolumes bool
var downYes bool

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove the project",
	Example: `  comquad down
  comquad down -d
  comquad down --dry-run
  comquad down -y`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !downYes && !dryRun && isStdinTerminal() {
			fmt.Printf("Are you sure you want to remove project %q? [y/N]: ", projectName)
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("aborted")
			}
		}

		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Down(downRemoveVolumes, dryRun)
	},
}

func isStdinTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func init() {
	downCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	downCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed without actually removing anything")
	downCmd.Flags().BoolVarP(&downRemoveVolumes, "delete-volumes", "d", false, "Remove named volumes declared in the compose file")
	downCmd.Flags().BoolVarP(&downYes, "yes", "y", false, "Skip confirmation prompt")
}
