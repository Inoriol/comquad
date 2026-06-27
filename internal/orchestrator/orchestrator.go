package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"comquad/internal/build"
	"comquad/internal/cooker"
	"comquad/internal/deploy"
	"comquad/internal/logger"
	"comquad/internal/preprocess"
	"comquad/internal/transpile"
)

// Orchestrator wires all internal packages together and drives
// the lifecycle of a comquad project.
type Orchestrator struct {
	projectName string
	cwd         string

	// newState and newSystemd are factories that create the state store and
	// systemd client respectively. They default to the real implementations
	// and can be overridden in tests to inject fakes.
	newState       func() (deploy.StateStore, error)
	newSystemd     func() (deploy.SystemdClient, error)
	listContainers func(projectName string, all bool) ([]ContainerInfo, error)
}

// NewOrchestrator creates a new Orchestrator for the current working directory.
func NewOrchestrator(projectName string) (*Orchestrator, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Fall back to directory name if no name provided
	if projectName == "" {
		projectName = filepath.Base(cwd)
	}

	return &Orchestrator{
		projectName: projectName,
		cwd:         cwd,
		newState: func() (deploy.StateStore, error) {
			return deploy.NewStateManager()
		},
		newSystemd: func() (deploy.SystemdClient, error) {
			return deploy.NewSystemdManager()
		},
		listContainers: func(projectName string, all bool) ([]ContainerInfo, error) {
			return listContainersFromPodman(projectName, all)
		},
	}, nil
}

// Up preprocesses, transpiles, cooks and deploys the project
// defined in the compose.yaml in the current working directory.
// If dryRun is true, the pipeline runs through the cook stage but nothing is
// written to the systemd directory, no state is registered, and no units are
// started — instead each generated quadlet file and its intended target path
// are printed to stdout.
// If follow is true, streams logs from the deployment timestamp onward.
func (o *Orchestrator) Up(forceBuild bool, pullStrategy string, follow bool, dryRun bool) error {
	composeFile := findComposeFile(o.cwd)
	if composeFile == "" {
		return fmt.Errorf("no compose file found in current directory (looked for compose.yaml, compose.yml, docker-compose.yaml, docker-compose.yml)")
	}

	targetDir, err := o.resolveTargetDir()
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "comquad-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Read compose file once to get build info
	composeData, err := os.ReadFile(composeFile)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}

	// Get build info before preprocessing
	buildInfo, err := o.getBuildInfo(composeData)
	if err != nil {
		return err
	}

	// Reuse composeData already read above — avoids a second disk read and
	// eliminates a TOCTOU window if the file changes between reads.
	processedYaml, err := o.preprocess(composeData)
	if err != nil {
		return err
	}

	if err := o.transpile(processedYaml, tempDir); err != nil {
		return err
	}

	isRootless := deploy.IsRootless()

	if dryRun {
		// In dry-run mode, cook into a dedicated preview dir inside the temp
		// directory so the real systemd target dir is never touched.
		previewDir, err := os.MkdirTemp("", "comquad-preview-*")
		if err != nil {
			return fmt.Errorf("failed to create preview directory: %w", err)
		}
		defer os.RemoveAll(previewDir)

		if err := o.cook(tempDir, previewDir, isRootless); err != nil {
			return err
		}

		projectFiles, err := o.collectProjectFiles(previewDir)
		if err != nil {
			return err
		}

		return o.printDryRun(projectFiles, previewDir, targetDir, buildInfo, pullStrategy)
	}

	if err := o.cook(tempDir, targetDir, isRootless); err != nil {
		return err
	}

	projectFiles, err := o.collectProjectFiles(targetDir)
	if err != nil {
		return err
	}

	// cleanup only removes files if registration fails
	// NOT on startUnits failure — files must stay for systemd
	cleanup := func() {
		for _, f := range projectFiles {
			os.Remove(f)
		}
		if dbusMgr, err := o.newSystemd(); err == nil {
			defer dbusMgr.Close()
			dbusMgr.ReloadDaemon(projectFiles...)
		}
	}

	if err := o.registerState(projectFiles); err != nil {
		cleanup()
		return err
	}

	// Handle images (build or pull) before starting units
	if err := o.handleImages(projectFiles, buildInfo, forceBuild, pullStrategy); err != nil {
		return err
	}

	// Capture deploy timestamp before starting units
	deployTime := time.Now().Format("2006-01-02 15:04:05")

	// No cleanup on startUnits failure — let systemd keep the files
	if err := o.startUnits(projectFiles); err != nil {
		return fmt.Errorf("units written but failed to start: %w", err)
	}

	logger.Success("Successfully deployed project: " + o.projectName)

	if follow {
		fmt.Println("Following logs for project:", o.projectName)
		return o.FollowLogs(deployTime, "", "")
	}

	return nil
}

// Down stops all units, removes quadlet files, removes networks, and unregisters the project.
// If removeVolumes is true, also removes Podman volumes.
func (o *Orchestrator) Down(removeVolumes bool) error {
	stateMgr, err := o.newState()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	state, exists := stateMgr.GetProject(o.projectName)
	if !exists {
		return fmt.Errorf("project %s is not deployed", o.projectName)
	}

	// Open a single D-Bus connection for the entire down sequence.
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	// Step 1: Stop all container units via systemd
	if err := o.stopUnits(dbusMgr, state.Files); err != nil {
		logger.Warn("Some units failed to stop: " + err.Error())
	}

	// Step 1a: Stop network units
	for _, f := range state.Files {
		if strings.HasSuffix(f, ".network") {
			unitName := NetworkFileToUnitName(f)
			logger.Print("Stopping unit: " + unitName)
			if err := dbusMgr.StopUnit(unitName); err != nil {
				logger.Warn("Failed to stop network unit " + unitName + ": " + err.Error())
			}
		}
	}

	// Step 1b: Stop volume units
	for _, f := range state.Files {
		if strings.HasSuffix(f, ".volume") {
			unitName := VolumeFileToUnitName(f)
			logger.Print("Stopping unit: " + unitName)
			if err := dbusMgr.StopUnit(unitName); err != nil {
				logger.Warn("Failed to stop volume unit " + unitName + ": " + err.Error())
			}
		}
	}

	// Step 2: Remove quadlet files from target dir
	for _, f := range state.Files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			logger.Warn("Failed to remove file " + f + ": " + err.Error())
		}
	}

	// Step 3: Reload daemon so systemd forgets the removed units and
	// releases its references to networks/volumes before we try to remove them.
	if err := dbusMgr.ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Step 4: Remove networks (after daemon-reload releases systemd references)
	if err := deploy.RemoveNetworks(o.projectName); err != nil {
		logger.Error("failed to remove networks: " + err.Error())
	}

	// Step 5: Remove volumes if requested
	if removeVolumes {
		if err := deploy.RemoveVolumes(o.projectName); err != nil {
			logger.Error("failed to remove volumes: " + err.Error())
		}
	}

	// Step 6: Unregister project from state
	if err := stateMgr.UnregisterProject(o.projectName); err != nil {
		return fmt.Errorf("failed to unregister project: %w", err)
	}

	logger.Success("Successfully removed project: " + o.projectName)
	return nil
}

// --- private helpers ---

func (o *Orchestrator) resolveTargetDir() (string, error) {
	resolver := deploy.NewTargetDirResolver()
	targetDir, err := resolver.GetSystemdPath()
	if err != nil {
		return "", fmt.Errorf("failed to resolve systemd path: %w", err)
	}
	return targetDir, nil
}

func (o *Orchestrator) preprocess(composeData []byte) ([]byte, error) {
	engine := preprocess.NewEngine(o.projectName, o.cwd)
	processed, err := engine.Process(composeData)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}

	return processed, nil
}

func (o *Orchestrator) getBuildInfo(composeData []byte) (map[string]*preprocess.BuildInfo, error) {
	engine := preprocess.NewEngine(o.projectName, o.cwd)
	return engine.GetBuildInfo(composeData)
}

func (o *Orchestrator) transpile(processedYaml []byte, tempDir string) error {
	podlet, err := transpile.NewPodletRunner(tempDir)
	if err != nil {
		return err
	}
	if err := podlet.Transpile(processedYaml); err != nil {
		return fmt.Errorf("transpilation failed: %w", err)
	}
	return nil
}

func (o *Orchestrator) cook(tempDir, targetDir string, isRootless bool) error {
	portOffset := 0
	if isRootless {
		portOffset = 2000
		if envOffset := os.Getenv("ROOTLESS_PORT_OFFSET"); envOffset != "" {
			if parsed, err := strconv.Atoi(envOffset); err == nil && parsed > 0 {
				portOffset = parsed
			}
		}
	}

	selinuxEnabled := preprocess.IsSELinuxEnabled()
	if selinuxEnabled {
		logger.Info(fmt.Sprintf("SELinux detected (%s), adding :z labels to all volumes", preprocess.SELinuxMode()))
	}

	cookerEngine := cooker.NewCooker(tempDir, targetDir, o.projectName, isRootless, portOffset, selinuxEnabled)
	if err := cookerEngine.Cook(); err != nil {
		return fmt.Errorf("cooking failed: %w", err)
	}
	return nil
}

func (o *Orchestrator) collectProjectFiles(targetDir string) ([]string, error) {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read target directory: %w", err)
	}

	var projectFiles []string
	prefix := "cq-" + o.projectName
	for _, f := range entries {
		if strings.HasPrefix(f.Name(), prefix) {
			projectFiles = append(projectFiles, filepath.Join(targetDir, f.Name()))
		}
	}

	return projectFiles, nil
}

func (o *Orchestrator) registerState(projectFiles []string) error {
	stateMgr, err := o.newState()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	return stateMgr.RegisterProject(deploy.ProjectState{
		ProjectName: o.projectName,
		SourcePath:  o.cwd,
		Files:       projectFiles,
	})
}

func (o *Orchestrator) startUnits(projectFiles []string) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	if err := dbusMgr.ReloadDaemon(projectFiles...); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".container") {
			unitName := containerFileToUnitName(f)
			logger.Action("Starting unit: " + unitName)

			if err := dbusMgr.WaitForUnit(unitName, 10*time.Second); err != nil {
				return fmt.Errorf("unit %s did not appear after daemon-reload: %w", unitName, err)
			}

			if err := dbusMgr.StartUnit(unitName); err != nil {
				return fmt.Errorf("failed to start unit %s: %w", unitName, err)
			}
		}
	}

	return nil
}

// handleImages builds or pulls images based on the compose file and strategy
func (o *Orchestrator) handleImages(projectFiles []string, buildInfo map[string]*preprocess.BuildInfo, forceBuild bool, pullStrategy string) error {
	// First handle build services
	for serviceName, info := range buildInfo {
		imageTag := build.GenerateBuildTag(o.projectName, serviceName)

		// Check if we need to build
		shouldBuild := forceBuild
		if !shouldBuild {
			engine := &build.Engine{}
			shouldBuild = !engine.ImageExists(imageTag)
		}

		if shouldBuild {
			engine := &build.Engine{}
			if err := engine.BuildService(
				serviceName,
				info.Context,
				info.Dockerfile,
				info.Args,
				info.Target,
				imageTag,
			); err != nil {
				return fmt.Errorf("failed to build image for service %s: %w", serviceName, err)
			}
			logger.Success("Built image: " + imageTag)
		} else {
			logger.Info("Image already exists locally, skipping build: " + imageTag)
		}
	}

	// Then handle image-only services
	imagePullStrategy, err := build.ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", f, err)
		}

		// Parse Image= line from the container file
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Image=") {
				image := strings.TrimPrefix(line, "Image=")
				image = strings.TrimSpace(image)

				// Skip build-generated images (they're already handled above)
				if strings.Contains(image, ":") {
					continue
				}

				engine := &build.Engine{
					PullStrategy: imagePullStrategy,
				}

				if err := engine.HandleImage("unknown", image); err != nil {
					return fmt.Errorf("failed to handle image %s: %w", image, err)
				}
				logger.Success("Handled image: " + image)
				break
			}
		}
	}

	return nil
}

// printDryRun prints a preview of what `comquad up` would deploy without
// writing anything to the real systemd directory or starting any units.
//
// projectFiles are paths inside previewDir. targetDir is the real systemd
// directory that would receive the files in a live run.
func (o *Orchestrator) printDryRun(
	projectFiles []string,
	previewDir string,
	targetDir string,
	buildInfo map[string]*preprocess.BuildInfo,
	pullStrategy string,
) error {
	fmt.Printf("Dry run — project: %s\n", o.projectName)
	fmt.Printf("Target directory: %s\n\n", targetDir)

	// --- Image actions ---
	imagePullStrategy, err := build.ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}

	// Build services
	for serviceName, info := range buildInfo {
		imageTag := build.GenerateBuildTag(o.projectName, serviceName)
		engine := &build.Engine{}
		if engine.ImageExists(imageTag) {
			fmt.Printf("[image] %-12s %s  (already exists locally, would skip build)\n", serviceName, imageTag)
		} else {
			fmt.Printf("[image] %-12s %s  (would build from %s)\n", serviceName, imageTag, info.Context)
		}
	}

	// Pull-only services — read Image= from the cooked container files
	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read preview file %s: %w", f, err)
		}
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "Image=") {
				continue
			}
			image := strings.TrimSpace(strings.TrimPrefix(line, "Image="))
			if strings.Contains(image, ":") {
				// build-generated image — already reported above
				break
			}
			switch imagePullStrategy {
			case build.PullAlways:
				fmt.Printf("[image] %-12s %s  (would pull: always)\n", filepath.Base(f), image)
			case build.PullMissing:
				engine := &build.Engine{}
				if engine.ImageExists(image) {
					fmt.Printf("[image] %-12s %s  (already exists locally, would skip pull)\n", filepath.Base(f), image)
				} else {
					fmt.Printf("[image] %-12s %s  (would pull: not found locally)\n", filepath.Base(f), image)
				}
			case build.PullNever:
				fmt.Printf("[image] %-12s %s  (pull skipped: never)\n", filepath.Base(f), image)
			}
			break
		}
	}

	if len(buildInfo) > 0 || len(projectFiles) > 0 {
		fmt.Println()
	}

	// --- Quadlet files ---
	fmt.Printf("%d quadlet file(s) would be written:\n\n", len(projectFiles))
	separator := strings.Repeat("─", 60)

	for _, f := range projectFiles {
		// Compute the target path: replace previewDir prefix with targetDir
		rel, err := filepath.Rel(previewDir, f)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}
		targetPath := filepath.Join(targetDir, rel)

		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read preview file %s: %w", f, err)
		}

		fmt.Printf("%s\n", separator)
		fmt.Printf("  %s\n", targetPath)
		fmt.Printf("%s\n", separator)
		fmt.Println(strings.TrimRight(string(content), "\n"))
		fmt.Println()
	}

	fmt.Println("Dry run complete — nothing was written, no units started.")
	return nil
}

func (o *Orchestrator) stopUnits(dbusMgr deploy.SystemdClient, projectFiles []string) error {
	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".container") {
			unitName := containerFileToUnitName(f)
			logger.Print("Stopping unit: " + unitName)
			if err := dbusMgr.StopUnit(unitName); err != nil {
				return fmt.Errorf("failed to stop unit %s: %w", unitName, err)
			}
		}
	}

	return nil
}

func (o *Orchestrator) verifyUnitsStopped(dbusMgr deploy.SystemdClient, projectFiles []string) error {
	var activeUnits []string
	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".container") {
			continue
		}
		unitName := containerFileToUnitName(f)
		units, err := dbusMgr.ListUnitsByNames([]string{unitName})
		if err != nil {
			continue
		}
		if len(units) > 0 && units[0].ActiveState == "active" {
			activeUnits = append(activeUnits, unitName)
		}
	}

	if len(activeUnits) > 0 {
		return fmt.Errorf("units still active after stop: %s", strings.Join(activeUnits, ", "))
	}

	return nil
}

// ContainerFileToUnitName derives the systemd unit name from a quadlet file path.
// e.g. /path/to/cq-myapp-web.container -> cq-myapp-web.service
func ContainerFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	return strings.TrimSuffix(base, ".container") + ".service"
}

// containerFileToUnitName is an unexported alias kept for internal call sites.
func containerFileToUnitName(filePath string) string {
	return ContainerFileToUnitName(filePath)
}

// NetworkFileToUnitName derives the systemd unit name from a network quadlet file path.
// e.g. /path/to/cq-myapp-default.network -> cq-myapp-default-network.service
func NetworkFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, ".network")
	return nameWithoutExt + "-network.service"
}

// VolumeFileToUnitName derives the systemd unit name from a volume quadlet file path.
// e.g. /path/to/cq-myapp-data.volume -> cq-myapp-data-volume.service
func VolumeFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, ".volume")
	return nameWithoutExt + "-volume.service"
}

func findComposeFile(dir string) string {
	candidates := []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
	}
	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}


