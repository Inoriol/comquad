package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
)

// Edit opens project units or a specific unit file in the editor.
// If noReload is false, systemd is reloaded and units are restarted after changes.
func (o *Orchestrator) Edit(projectArg string, noReload bool) error {
	_, state, err := o.ensureProjectDeployed()
	if err != nil {
		return err
	}

	if projectArg == "" {
		return o.editProject(state, noReload)
	}

	return o.editUnit(state, projectArg, noReload)
}

func (o *Orchestrator) editProject(state deploy.ProjectState, noReload bool) error {
	files := o.resolveAllProjectFiles(state)
	if len(files) == 0 {
		return fmt.Errorf("no units found for project %s", o.projectName)
	}

	return o.openAndReload(files, noReload)
}

func (o *Orchestrator) editUnit(state deploy.ProjectState, arg string, noReload bool) error {
	var found string

	if found = MatchFirstContainer(o.projectName, state, arg); found != "" {
		// do nothing
	} else if found = MatchQuadletResource(o.projectName, state, arg); found != "" {
		// do nothing
	}

	if found == "" {
		var suggestions []string
		for _, f := range state.Files {
			suggestions = append(suggestions, filepath.Base(f))
		}
		return fmt.Errorf("no unit found for '%s', did you mean: %s", arg, strings.Join(suggestions, ", "))
	}

	return o.openAndReload([]string{found}, noReload)
}

func (o *Orchestrator) resolveAllProjectFiles(state deploy.ProjectState) []string {
	var files []string
	for _, f := range state.Files {
		if strings.HasSuffix(f, ".container") || strings.HasSuffix(f, ".network") ||
			strings.HasSuffix(f, ".volume") || strings.HasSuffix(f, ".image") ||
			strings.HasSuffix(f, ".build") || strings.HasSuffix(f, ".cq-secret") ||
			strings.HasSuffix(f, ".cq-image") || strings.HasSuffix(f, ".cq-build") {
			files = append(files, f)
		}
	}
	return files
}

func (o *Orchestrator) openAndReload(files []string, noReload bool) error {
	// Read original content
	originals := make(map[string]string)
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", f, err)
		}
		originals[f] = string(content)
	}

	editorEnv := os.Getenv("EDITOR")
	if editorEnv == "" {
		editorEnv = findDefaultEditor()
	}

	// Split $EDITOR on whitespace so that values like "vim -o" or "code --wait" work.
	editorParts := strings.Fields(editorEnv)
	editorBin := editorParts[0]
	editorArgs := append(editorParts[1:], files...)

	if _, err := exec.LookPath(editorBin); err != nil {
		return fmt.Errorf("editor %q not found in PATH: %w", editorBin, err)
	}

	// Build editor command with all files
	cmd := exec.Command(editorBin, editorArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Check for changes
	var changedFiles []string
	for _, f := range files {
		newContent, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read %s after edit: %w", f, err)
		}
		if string(newContent) != originals[f] {
			changedFiles = append(changedFiles, f)
		}
	}

	if len(changedFiles) == 0 {
		logger.Print("No changes detected.")
		return nil
	}

	logger.Printf("Changes detected in %d file(s):\n", len(changedFiles))
	for _, f := range changedFiles {
		logger.Printf("  %s\n", f)
	}

	if noReload {
		logger.Print("Skipping reload (--no-reload flag set).")
		return nil
	}

	// Copy pseudo-quadlet files to their .service counterparts
	var reloadFiles []string
	reloadFiles = append(reloadFiles, changedFiles...)
	for _, f := range changedFiles {
		base := filepath.Base(f)
		var serviceName string
		switch {
		case strings.HasSuffix(base, ".cq-secret"):
			serviceName = strings.TrimSuffix(base, ".cq-secret") + "-secret.service"
		case strings.HasSuffix(base, ".cq-image"):
			serviceName = strings.TrimSuffix(base, ".cq-image") + "-image.service"
		case strings.HasSuffix(base, ".cq-build"):
			serviceName = strings.TrimSuffix(base, ".cq-build") + "-build.service"
		default:
			continue
		}
		servicePath := filepath.Join(filepath.Dir(f), serviceName)
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", f, err)
		}
		if err := os.WriteFile(servicePath, content, 0644); err != nil {
			return fmt.Errorf("failed to write service file %s: %w", servicePath, err)
		}
		reloadFiles = append(reloadFiles, servicePath)
		logger.Printf("Updated service file: %s\n", serviceName)
	}

	// Reload systemd daemon
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	logger.Print("Reloading systemd daemon...")
	if err := dbusMgr.ReloadDaemon(reloadFiles...); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Restart changed container units
	var restartCount int
	var failedCount int
	for _, f := range changedFiles {
		if strings.HasSuffix(f, ".container") {
			unitName := ContainerFileToUnitName(f)
			logger.Printf("Restarting unit: %s\n", unitName)
			if err := dbusMgr.RestartUnit(unitName); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to restart unit %s: %v\n", unitName, err)
				failedCount++
			} else {
				restartCount++
			}
		}
	}

	// Restart changed pseudo-quadlet services
	for _, f := range changedFiles {
		base := filepath.Base(f)
		var unitName string
		switch {
		case strings.HasSuffix(base, ".cq-secret"):
			unitName = CQSecretFileToUnitName(f)
		case strings.HasSuffix(base, ".cq-image"):
			unitName = CQImageFileToUnitName(f)
		case strings.HasSuffix(base, ".cq-build"):
			unitName = CQBuildFileToUnitName(f)
		default:
			continue
		}
		logger.Printf("Restarting unit: %s\n", unitName)
		if err := dbusMgr.RestartUnit(unitName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to restart unit %s: %v\n", unitName, err)
			failedCount++
		} else {
			restartCount++
		}
	}

	msg := fmt.Sprintf("Reloaded daemon, restarted %d unit(s)", restartCount)
	if failedCount > 0 {
		msg += fmt.Sprintf(", %d failed", failedCount)
	}
	logger.Print(msg)
	return nil
}

func findDefaultEditor() string {
	for _, editor := range []string{"editor", "nano", "vim", "vi"} {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}
	return "vi"
}
