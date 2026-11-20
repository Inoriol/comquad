package main

import (
	"fmt"

	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/orchestrator"
	"github.com/Inoriol/comquad/internal/output"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run JOB",
	Short: "Run a job manually",
	Long: `Run a job defined in compose.yaml manually.

Jobs are one-shot containers that run to completion. Unlike services which
are long-running, jobs execute a task and then exit.

A job can be triggered manually regardless of whether it also has automated
triggers (schedules) configured, unless the job has manual execution
explicitly disabled with 'manual: false'.

Examples:
  comquad run db-migration
  comquad run backup --follow
  comquad run cleanup --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, _ := cmd.Flags().GetString("project-name")
		follow, _ := cmd.Flags().GetBool("follow")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		orch, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}

		runOpts := orchestrator.RunOptions{
			JobName: args[0],
			Follow:  follow,
			DryRun:  dryRun,
		}

		if err := orch.Run(runOpts); err != nil {
			if output.IsJSONMode() {
				return output.PrintError("run_failed", err.Error())
			}
			return err
		}

		if !output.IsJSONMode() && !follow {
			logger.Success(fmt.Sprintf("Job %s completed", args[0]))
		}

		return nil
	},
}

func init() {
	runCmd.Flags().BoolP("follow", "f", false, "Follow job logs after starting")
	runCmd.Flags().Bool("dry-run", false, "Preview without running the job")
	rootCmd.AddCommand(runCmd)
}
