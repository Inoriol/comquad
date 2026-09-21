package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var buildPullStrategy string
var buildFollow bool
var buildNoDiff bool

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build images defined in compose.yaml without starting containers",
	Example: `  comquad build
  comquad build -f                                 # Follow build logs
  comquad build --pull always                      # Always pull base images
  comquad build --dry-run                          # Preview without building
  comquad build --no-diff                          # Apply changes without showing a diff
  comquad build -n my-project                      # Override project name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}

		pullStr := buildPullStrategy
		if pullStr == "" {
			pullStr = "always"
		}

		return o.Build(pullStr, buildFollow, dryRun, buildNoDiff)
	},
}

func init() {
	buildCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	buildCmd.Flags().StringVarP(&buildPullStrategy, "pull", "p", "always", "Image pull strategy: 'always' (default), 'missing', or 'never'")
	buildCmd.Flags().BoolVarP(&buildFollow, "follow", "f", false, "Follow build logs")
	buildCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview generated quadlet files without writing or building anything")
	buildCmd.Flags().BoolVar(&buildNoDiff, "no-diff", false, "Apply changes without showing a diff or asking for confirmation")
}
