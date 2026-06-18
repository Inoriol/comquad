package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"comquad/internal/deploy"
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
		return o.Down(downRemoveVolumes)
	},
}

var downRemoveVolumes bool

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
	Short: "Check that required tools and services are available",
	RunE: func(cmd *cobra.Command, args []string) error {
		tools := []string{"podman", "podlet"}
		var missing []string
		var warnings []string

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

		if _, err := exec.LookPath("systemctl"); err != nil {
			warnings = append(warnings, "systemctl not found — systemd integration may not work")
		}

		sm, err := deploy.NewSystemdManager()
		if err != nil {
			warnings = append(warnings, "D-Bus connection failed: "+err.Error())
		} else {
			sm.Close()
			fmt.Println("  D-Bus: connected")
		}

		resolver := deploy.NewTargetDirResolver()
		targetDir, err := resolver.GetSystemdPath()
		if err == nil {
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				warnings = append(warnings, "target directory not writable: "+targetDir)
			} else {
				fmt.Printf("  Target dir: %s (writable)\n", targetDir)
			}
		}

		if len(warnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, w := range warnings {
				fmt.Printf("  - %s\n", w)
			}
		} else {
			fmt.Println("\nAll system checks passed.")
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
var dryRun bool

var regenerateCmd = &cobra.Command{
	Use:   "regenerate",
	Short: "Restore state from Podman labels",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !force {
			return fmt.Errorf("regenerate requires --force to overwrite existing state")
		}
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}
		return o.Regenerate()
	},
}

var force bool

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
	downCmd.Flags().BoolVarP(&downRemoveVolumes, "volumes", "v", false, "Remove named volumes declared in the compose file")
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
	regenerateCmd.Flags().BoolVar(&force, "force", false, "Force regeneration of state file")
	regenerateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be regenerated without writing")
}

func main() {
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(regenerateCmd)
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
