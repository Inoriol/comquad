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
		return o.Up()
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

func init() {
	upCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	downCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	listCmd.Flags().StringVarP(&projectName, "name", "n", "", "Filter by project name")
}

func main() {
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(checkCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
