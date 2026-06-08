package transpile

import (
	"bytes"
	"fmt"
	"os/exec"
)

// PodletRunner handles the execution of the podlet command
type PodletRunner struct {
	PodletPath string
	TempDir    string
}

// NewPodletRunner creates a new runner
func NewPodletRunner(tempDir string) *PodletRunner {
	return &PodletRunner{
		PodletPath: "podlet", // Assumes podlet is in PATH
		TempDir:    tempDir,
	}
}

// Transpile takes the processed YAML and runs podlet against it
func (r *PodletRunner) Transpile(input []byte) error {
	// We use -f to specify the output directory for quadlet files
	cmd := exec.Command(r.PodletPath, "-f", r.TempDir, "compose")

	// Create a buffer to pipe stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start podlet: %w", err)
	}

	// Write the YAML to podlet's stdin
	if _, err := stdin.Write(input); err != nil {
		return fmt.Errorf("failed to write to podlet stdin: %w", err)
	}

	if err := stdin.Close(); err != nil {
		return fmt.Errorf("failed to close podlet stdin: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("podlet execution failed: %s: %w", stderr.String(), err)
	}

	return nil
}
