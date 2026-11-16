package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var composePsAll bool

var composePsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(resolveComposeProjectName())
		if err != nil {
			return err
		}
		if composeFile != "" {
			o.WithComposeFile(composeFile)
		}
		return o.Ps(composePsAll)
	},
}

func init() {
	composePsCmd.Flags().BoolVarP(&composePsAll, "all", "a", false, "Show all containers including exited ones")
}
