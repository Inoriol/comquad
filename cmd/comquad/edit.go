package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var noReload bool

var editCmd = &cobra.Command{
	Use:   "edit [service]",
	Short: "Edit systemd units for a project or a specific unit file",
	Example: `  comquad edit web          # Edit cq-myapp-web.container
  comquad edit                # Edit all project units
  comquad edit --no-reload    # Edit without auto-reloading systemd`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}

		var projectArg string
		if len(args) > 0 {
			projectArg = args[0]
		}

		return o.Edit(projectArg, noReload)
	},
}

func init() {
	editCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	editCmd.Flags().BoolVar(&noReload, "no-reload", false, "Open files in editor without reloading systemd")
}
