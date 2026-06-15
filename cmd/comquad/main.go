package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"comquad/internal/orchestrator"
)

var rootCmd = &cobra.Command{
	Use:   "comquad",
	Short: "comquad is a developer-friendly CLI for deploying Podman Quadlets.",
}

var projectName string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Deploy the project defined in compose.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}

		pullStr := pullStrategy
		if pullStr == "" {
			pullStr = "missing"
		}

		return o.Up(forceBuild, pullStr, upFollow)
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Down()
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all currently deployed projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.List()
	},
}

var follow bool
var upFollow bool
var forceBuild bool
var pullStrategy string
var noReload bool

var logsCmd = &cobra.Command{
	Use:   "logs [service ...]",
	Short: "Print logs for a deployed project",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		follow, _ := cmd.Flags().GetBool("follow")
		return o.Logs(args, follow)
	},
}

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "Show the current state of project units",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Ps()
	},
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check that required tools are available",
	RunE: func(cmd *cobra.Command, args []string) error {
		tools := []string{"podman", "podlet"}
		var missing []string
		for _, tool := range tools {
			if _, err := exec.LookPath(tool); err != nil {
				missing = append(missing, tool)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
		}
		fmt.Println("All required tools are available:")
		for _, tool := range tools {
			path, _ := exec.LookPath(tool)
			fmt.Printf("  %s: %s\n", tool, path)
		}
		return nil
	},
}

var viewCmd = &cobra.Command{
	Use:   "view [project] [service]",
	Short: "View systemd units for a project or display a specific unit file",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}

		var projectArg string
		if len(args) > 0 {
			projectArg = args[0]
		}

		return o.View(projectArg)
	},
}

var editCmd = &cobra.Command{
	Use:   "edit [project] [service]",
	Short: "Edit systemd units for a project or a specific unit file",
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

var startCmd = &cobra.Command{
	Use:   "start [service ...]",
	Short: "Start all units for the project or specific services",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Start(args)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop [service ...]",
	Short: "Stop all units for the project or specific services",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Stop(args)
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart [service ...]",
	Short: "Restart all units for the project or specific services",
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Restart(args)
	},
}

var execUser string
var execTTY bool

var execCmd = &cobra.Command{
	Use:   "exec [service] <command...>",
	Short: "Run a command inside a running container",
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
	upCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	upCmd.Flags().BoolVarP(&forceBuild, "build", "b", false, "Force rebuild images even if they exist locally")
	upCmd.Flags().StringVarP(&pullStrategy, "pull", "p", "missing", "Image pull strategy: 'always', 'missing' (default), or 'never'")
	upCmd.Flags().BoolVarP(&upFollow, "follow", "f", false, "Follow logs after deployment")
	downCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	listCmd.Flags().StringVarP(&projectName, "name", "n", "", "Filter by project name")
	logsCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	psCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	viewCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	editCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	editCmd.Flags().BoolVar(&noReload, "no-reload", false, "Open files in editor without reloading systemd")
	startCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	stopCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	restartCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	execCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	execCmd.Flags().StringVarP(&execUser, "user", "u", "", "User to run as inside the container")
	execCmd.Flags().BoolVarP(&execTTY, "tty", "t", true, "Allocate a TTY (default: true)")
}

func main() {
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(psCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(execCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
