package main

import (
	"os"
	"path/filepath"
	"strings"
)

func isProviderMode() bool {
	if len(os.Args) == 0 {
		return false
	}
	binaryName := filepath.Base(os.Args[0])
	binaryName = strings.TrimSuffix(binaryName, filepath.Ext(binaryName))
	if binaryName == "docker-compose" || binaryName == "podman-compose" {
		return true
	}
	if binaryName == "comquad" && len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "compose" {
				return false
			}
		}
		composeCommands := map[string]bool{
			"up": true, "down": true, "ps": true, "logs": true,
			"start": true, "stop": true, "restart": true, "exec": true,
			"build": true, "pull": true,
		}
		for _, arg := range os.Args[1:] {
			if composeCommands[arg] {
				return true
			}
		}
	}
	return false
}
