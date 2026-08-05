package main

import (
	"github.com/spf13/cobra"

	"comquad/internal/orchestrator"
)

var psAll bool

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "Show the current state of project units",
	Example: `  comquad ps
  comquad ps -a     # Include exited containers`,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Ps(psAll)
	},
}

func init() {
	psCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	psCmd.Flags().BoolVarP(&psAll, "all", "a", false, "Show all containers including exited ones")
}
