package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/orchestrator"
)

var pullStrategy string
var upFollow bool

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Deploy the project defined in compose.yaml",
	Example: `  comquad up
  comquad up -f                                  # Follow logs after deployment
  comquad up --pull always                       # Always pull images from registry
  comquad up --dry-run                           # Preview without deploying
  comquad up -n my-project                       # Override project name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := orchestrator.NewOrchestrator(projectName)
		if err != nil {
			return err
		}

		pullStr := pullStrategy
		if pullStr == "" {
			pullStr = "missing"
		}

		return o.Up(pullStr, upFollow, dryRun)
	},
}

func init() {
	upCmd.Flags().StringVarP(&projectName, "name", "n", "", "Override project name (default: current directory name)")
	upCmd.Flags().StringVarP(&pullStrategy, "pull", "p", "missing", "Image pull strategy: 'always', 'missing' (default), or 'never'")
	upCmd.Flags().BoolVarP(&upFollow, "follow", "f", false, "Follow logs after deployment")
	upCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview generated quadlet files without writing or starting anything")
}
