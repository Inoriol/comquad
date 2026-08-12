package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check that required tools and services are available",
	RunE: func(cmd *cobra.Command, args []string) error {
		tools := []string{"podman"}
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

		logger.Print("All required tools are available:")
		for _, tool := range tools {
			path, _ := exec.LookPath(tool)
			logger.Printf("  %s: %s\n", tool, path)
		}

		if err := deploy.ValidatePodmanVersion(); err != nil {
			warnings = append(warnings, err.Error())
		} else {
			out, _ := exec.Command("podman", "version", "--format", "{{.Version}}").Output()
			logger.Printf("  Podman version: %s\n", strings.TrimSpace(string(out)))
		}

		if _, err := exec.LookPath("systemctl"); err != nil {
			warnings = append(warnings, "systemctl not found — systemd integration may not work")
		}

		sm, err := deploy.NewSystemdManager()
		if err != nil {
			warnings = append(warnings, "D-Bus connection failed: "+err.Error())
		} else {
			sm.Close()
			logger.Print("  D-Bus: connected")
		}

		resolver := deploy.NewTargetDirResolver()
		targetDir, err := resolver.GetSystemdPath()
		if err == nil {
			probeDir := targetDir
			if _, statErr := os.Stat(targetDir); os.IsNotExist(statErr) {
				probeDir = filepath.Dir(targetDir)
			}
			tmp, writeErr := os.CreateTemp(probeDir, ".comquad-check-*")
			if writeErr != nil {
				warnings = append(warnings, "target directory not writable: "+targetDir)
			} else {
				tmp.Close()
				os.Remove(tmp.Name())
				logger.Printf("  Target dir: %s (writable)\n", targetDir)
			}
		}

		if len(warnings) > 0 {
			logger.Print("\nWarnings:")
			for _, w := range warnings {
				logger.Printf("  - %s\n", w)
			}
		} else {
			logger.Print("\nAll system checks passed.")
		}

		return nil
	},
}
