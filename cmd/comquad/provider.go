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
	if os.Getenv("PODMAN_COMPOSE_PROVIDER") == "comquad" {
		return true
	}
	return false
}
