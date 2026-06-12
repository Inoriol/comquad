package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"comquad/internal/deploy"
)

// Edit opens project units or a specific unit file in the editor.
// If noReload is false, systemd is reloaded and units are restarted after changes.
func (o *Orchestrator) Edit(projectArg string, noReload bool) error {
	stateMgr, err := deploy.NewStateManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	state, exists := stateMgr.Projects[o.projectName]
	if !exists {
		return fmt.Errorf("project %s is not deployed", o.projectName)
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

	if found = MatchContainer(o.projectName, state, arg); found != "" {
		// do nothing
	} else if found = MatchNetworkOrVolume(o.projectName, state, arg); found != "" {
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
		if strings.HasSuffix(f, ".container") || strings.HasSuffix(f, ".network") || strings.HasSuffix(f, ".volume") {
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

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Build editor command with all files
	cmd := exec.Command(editor)
	cmd.Args = append(cmd.Args, files...)
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
		fmt.Println("No changes detected.")
		return nil
	}

	fmt.Printf("Changes detected in %d file(s):\n", len(changedFiles))
	for _, f := range changedFiles {
		fmt.Printf("  %s\n", f)
	}

	if noReload {
		fmt.Println("Skipping reload (--no-reload flag set).")
		return nil
	}

	// Reload systemd daemon
	dbusMgr, err := deploy.NewSystemdManager()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	fmt.Println("Reloading systemd daemon...")
	if err := dbusMgr.ReloadDaemon(changedFiles...); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Restart changed container units
	var restartCount int
	for _, f := range changedFiles {
		if strings.HasSuffix(f, ".container") {
			unitName := containerFileToUnitName(f)
			fmt.Printf("Restarting unit: %s\n", unitName)
			if err := dbusMgr.RestartUnit(unitName); err != nil {
				fmt.Printf("Warning: failed to restart unit %s: %v\n", unitName, err)
			} else {
				restartCount++
			}
		}
	}

	fmt.Printf("Reloaded daemon, restarted %d container unit(s)\n", restartCount)
	return nil
}
