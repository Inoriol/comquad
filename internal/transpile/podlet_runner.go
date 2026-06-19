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

// NewPodletRunner creates a new runner. It resolves the podlet binary in PATH
// immediately so callers get a clear error before the pipeline starts rather
// than a confusing failure deep in Transpile.
func NewPodletRunner(tempDir string) (*PodletRunner, error) {
	path, err := exec.LookPath("podlet")
	if err != nil {
		return nil, fmt.Errorf("podlet not found in PATH: %w", err)
	}
	return &PodletRunner{
		PodletPath: path,
		TempDir:    tempDir,
	}, nil
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
		stdin.Close() // prevent pipe leak if Start fails
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
