package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var composeUpDetach bool
var composeUpBuild bool
var composeUpNoBuild bool
var composeUpForceRecreate bool
var composeUpNoRecreate bool
var composeUpRemoveOrphans bool
var composeUpPull string

var composeUpCmd = &cobra.Command{
	Use:   "up [service...]",
	Short: "Create and start containers",
	Long: `Builds, (re)creates, starts, and attaches to containers for a service.

Without -d, attaches to container logs (like docker compose up).
With -d, starts containers in background and returns to shell.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(resolveComposeProjectName())
		if err != nil {
			return err
		}
		if composeFile != "" {
			o.WithComposeFile(composeFile)
		}

		pullStr := composeUpPull
		if pullStr == "" {
			pullStr = "missing"
		}

		return o.Up(orchestrator.UpOptions{
			PullStrategy:  pullStr,
			Follow:        !composeUpDetach,
			DryRun:        dryRun,
			NoDiff:        true,
			ForceRebuild:  composeUpBuild,
			SkipBuild:     composeUpNoBuild,
			ForceRecreate: composeUpForceRecreate,
			SkipRecreate:  composeUpNoRecreate,
			RemoveOrphans: composeUpRemoveOrphans,
			Services:      args,
		})
	},
}

func init() {
	composeUpCmd.Flags().BoolVarP(&composeUpDetach, "detach", "d", false, "Detached mode: run containers in the background")
	composeUpCmd.Flags().BoolVar(&composeUpBuild, "build", false, "Build images before starting containers")
	composeUpCmd.Flags().BoolVar(&composeUpNoBuild, "no-build", false, "Don't build an image, even if it's policy")
	composeUpCmd.Flags().BoolVar(&composeUpForceRecreate, "force-recreate", false, "Recreate containers even if their configuration hasn't changed")
	composeUpCmd.Flags().BoolVar(&composeUpNoRecreate, "no-recreate", false, "Don't recreate existing containers")
	composeUpCmd.Flags().BoolVar(&composeUpRemoveOrphans, "remove-orphans", false, "Remove containers for services not defined in the Compose file")
	composeUpCmd.Flags().StringVar(&composeUpPull, "pull", "missing", "Pull image before running (always|missing|never)")
}
