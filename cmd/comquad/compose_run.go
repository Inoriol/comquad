package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var composeRunDetach bool
var composeRunFollow bool

var composeRunCmd = &cobra.Command{
	Use:   "run JOB [COMMAND...]",
	Short: "Run a job manually",
	Long: `Run a job defined in compose.yaml manually.

This is the docker-compose compatible equivalent of 'comquad run'.
Jobs are one-shot containers that run to completion.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("requires a job name")
		}

		o, err := orchestrator.NewOrchestrator(resolveComposeProjectName())
		if err != nil {
			return err
		}
		if composeFile != "" {
			o.WithComposeFile(composeFile)
		}

		follow := composeRunFollow
		if composeRunDetach {
			follow = false
		}

		runOpts := orchestrator.RunOptions{
			JobName: args[0],
			Follow:  follow,
			DryRun:  dryRun,
		}

		return o.Run(runOpts)
	},
}

func init() {
	composeRunCmd.Flags().BoolVarP(&composeRunDetach, "detach", "d", false, "Run in background")
	composeRunCmd.Flags().BoolVar(&composeRunFollow, "follow", false, "Follow job logs")
}
