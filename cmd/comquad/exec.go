package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"comquad/internal/orchestrator"
)

var execUser string
var execTTY bool

var execCmd = &cobra.Command{
	Use:   "exec [service] <command...>",
	Short: "Run a command inside a running container",
	Example: `  comquad exec web ls /app
  comquad exec web sh
  comquad exec -u root web bash
  comquad exec -t=false web cat /etc/hosts`,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}

		var service string
		var command []string

		if len(args) > 0 {
			service = args[0]
			command = args[1:]
		}

		if len(command) == 0 {
			return fmt.Errorf("exec requires a command to run: comquad exec [service] <command>")
		}

		return o.Exec(service, execUser, execTTY, command)
	},
}

func init() {
	execCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	execCmd.Flags().StringVarP(&execUser, "user", "u", "", "User to run as inside the container")
	execCmd.Flags().BoolVarP(&execTTY, "tty", "t", true, "Allocate a TTY (default: true)")
}
