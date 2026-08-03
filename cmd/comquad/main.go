package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"comquad/internal/logger"
)

var quiet bool
var verbose bool
var projectName string
var dryRun bool

var rootCmd = &cobra.Command{
	Use:   "comquad",
	Short: "comquad is a developer-friendly CLI for deploying Podman Quadlets.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.SetQuiet(quiet)
		logger.SetVerbose(verbose)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all non-error output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information about changes made during deployment")
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
