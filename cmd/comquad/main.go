package main

import (
        "fmt"
        "os"

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
                // List doesn't need a project name or cwd — it reads all state
                o, err := orchestrator.NewOrchestrator("")
                if err != nil {
                        return err
                }
                return o.List()
        },
}

func init() {
        upCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
        downCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
        // list doesn't need --name since it shows all projects
}

func main() {
        rootCmd.AddCommand(upCmd)
        rootCmd.AddCommand(downCmd)
        rootCmd.AddCommand(listCmd)

        if err := rootCmd.Execute(); err != nil {
                fmt.Fprintln(os.Stderr, err)
                os.Exit(1)
        }
}
