package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var composeBuildPull string
var composeBuildFollow bool

var composeBuildCmd = &cobra.Command{
	Use:   "build [service...]",
	Short: "Build or rebuild services",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(resolveComposeProjectName())
		if err != nil {
			return err
		}
		if composeFile != "" {
			o.WithComposeFile(composeFile)
		}

		pullStr := composeBuildPull
		if pullStr == "" {
			pullStr = "always"
		}

		return o.Build(orchestrator.UpOptions{
			PullStrategy: pullStr,
			Follow:       composeBuildFollow,
			DryRun:       dryRun,
			NoDiff:       true,
			Services:     args,
		})
	},
}

func init() {
	composeBuildCmd.Flags().StringVar(&composeBuildPull, "pull", "always", "Pull image before building (always|missing|never)")
	composeBuildCmd.Flags().BoolVarP(&composeBuildFollow, "follow", "f", false, "Follow build logs")
}
