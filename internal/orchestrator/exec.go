package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/logger"
)

// Exec runs a command inside a single running container via podman exec.
func (o *Orchestrator) Exec(service string, user string, tty bool, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("exec requires a command to run")
	}

	_, state, err := o.ensureProjectDeployed()
	if err != nil {
		return err
	}

	// Resolve service to container quadlet files
	var matches []string
	if service != "" {
		matches = MatchAllContainers(o.projectName, state, service)
		if len(matches) == 0 {
			if MatchNetworkOrVolume(o.projectName, state, service) != "" {
				return fmt.Errorf("cannot exec into network or volume '%s'", service)
			}
			return fmt.Errorf("no containers found matching '%s' for project '%s'", service, o.projectName)
		}
	} else {
		for _, f := range state.Files {
			if strings.HasSuffix(f, ".container") {
				matches = append(matches, f)
			}
		}
	}

	if len(matches) == 0 {
		return fmt.Errorf("no containers found for project '%s'", o.projectName)
	}

	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, f := range matches {
			base := strings.TrimSuffix(filepath.Base(f), ".container")
			names = append(names, strings.TrimPrefix(base, "cq-"))
		}
		return fmt.Errorf("ambiguous service '%s', matched multiple containers: %s. Please specify a single container", service, strings.Join(names, ", "))
	}

	// Derive container name from the base filename: cq-myapp-web.container -> myapp-web
	base := strings.TrimSuffix(filepath.Base(matches[0]), ".container")
	containerName := strings.TrimPrefix(base, "cq-")

	if _, err := exec.LookPath("podman"); err != nil {
		return fmt.Errorf("podman not found in PATH: %w", err)
	}

	// Build podman exec command
	cmdArgs := []string{"exec"}
	if tty {
		cmdArgs = append(cmdArgs, "-t")
	}
	if user != "" {
		cmdArgs = append(cmdArgs, "-u", user)
	}
	cmdArgs = append(cmdArgs, containerName)
	cmdArgs = append(cmdArgs, command...)

	cmd := exec.Command("podman", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Printf("Executing in container '%s': podman exec %s\n", containerName, strings.Join(cmdArgs[2:], " "))

	return cmd.Run()
}
