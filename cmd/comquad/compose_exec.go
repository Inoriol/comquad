package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var composeExecUser string
var composeExecTTY bool
var composeExecNoTty bool

var composeExecCmd = &cobra.Command{
	Use:   "exec [service] <command...>",
	Short: "Execute a command in a running container",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(resolveComposeProjectName())
		if err != nil {
			return err
		}
		if composeFile != "" {
			o.WithComposeFile(composeFile)
		}

		var service string
		var command []string

		if len(args) > 0 {
			service = args[0]
			command = args[1:]
		}

		if len(command) == 0 {
			return fmt.Errorf("exec requires a command to run: comquad compose exec [service] <command>")
		}

		tty := composeExecTTY
		if composeExecNoTty {
			tty = false
		}

		return o.Exec(service, composeExecUser, tty, command)
	},
}

func init() {
	composeExecCmd.Flags().StringVarP(&composeExecUser, "user", "u", "", "User to run as inside the container")
	composeExecCmd.Flags().BoolVarP(&composeExecTTY, "tty", "t", true, "Allocate a TTY")
	composeExecCmd.Flags().BoolVarP(&composeExecNoTty, "no-TTY", "T", false, "Disable pseudo-TTY allocation")
}
