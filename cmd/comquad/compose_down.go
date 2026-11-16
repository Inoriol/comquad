package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var composeDownVolumes bool
var composeDownRemoveOrphans bool

var composeDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove containers, networks",
	Long: `Stops containers and removes containers, networks created by 'up'.

By default, named volumes are not removed. Use -v to remove them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(resolveComposeProjectName())
		if err != nil {
			return err
		}
		if composeFile != "" {
			o.WithComposeFile(composeFile)
		}

		return o.Down(orchestrator.DownOptions{
			RemoveVolumes: composeDownVolumes,
			DryRun:        dryRun,
			RemoveOrphans: composeDownRemoveOrphans,
		})
	},
}

func init() {
	composeDownCmd.Flags().BoolVar(&composeDownVolumes, "volumes", false, "Remove named volumes declared in the compose file")
	composeDownCmd.Flags().BoolVar(&composeDownRemoveOrphans, "remove-orphans", false, "Remove containers for services not defined in the Compose file")
}
