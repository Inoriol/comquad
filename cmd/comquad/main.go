package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/output"
)

var version = "dev"

var quiet bool
var verbose bool
var jsonOutput bool
var projectName string
var dryRun bool

var rootCmd = &cobra.Command{
	Use:   "comquad",
	Short: "comquad is a developer-friendly CLI for deploying Podman Quadlets.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.SetQuiet(quiet)
		logger.SetVerbose(verbose)
		output.SetJSONMode(jsonOutput)
	},
}

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
	rootCmd.Version = version
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all non-error output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information about changes made during deployment")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output results in JSON format")
}

func main() {
	if isProviderMode() && len(os.Args) > 1 {
		newArgs := []string{os.Args[0], "compose"}
		newArgs = append(newArgs, os.Args[1:]...)
		os.Args = newArgs
	}

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(pullCmd)
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
	rootCmd.AddCommand(composeCmd)

	if err := rootCmd.Execute(); err != nil {
		if output.IsJSONMode() {
			output.PrintError("COMMAND_FAILED", err.Error())
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
