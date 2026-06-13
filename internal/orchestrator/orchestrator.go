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
	"comquad/internal/preprocess"
	"comquad/internal/transpile"
)

// Orchestrator wires all internal packages together and drives
// the lifecycle of a comquad project.
type Orchestrator struct {
	projectName string
	cwd         string
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
	}, nil
}

// Up preprocesses, transpiles, cooks and deploys the project
// defined in the compose.yaml in the current working directory.
// If follow is true, streams logs from the deployment timestamp onward.
func (o *Orchestrator) Up(forceBuild bool, pullStrategy string, follow bool) error {
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

	processedYaml, err := o.preprocess(composeFile)
	if err != nil {
		return err
	}

	if err := o.transpile(processedYaml, tempDir); err != nil {
		return err
	}

	isRootless := isRootless()
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
		if dbusMgr, err := deploy.NewSystemdManager(); err == nil {
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

	fmt.Println("Successfully deployed project:", o.projectName)

	if follow {
		fmt.Println("Following logs for project:", o.projectName)
		return o.FollowLogs(deployTime)
	}

	return nil
}

// Down stops all units, removes quadlet files and unregisters the project.
func (o *Orchestrator) Down() error {
	stateMgr, err := deploy.NewStateManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	state, exists := stateMgr.Projects[o.projectName]
	if !exists {
		return fmt.Errorf("project %s is not deployed", o.projectName)
	}

	// Step 1: Stop all units via systemd
	if err := o.stopUnits(state.Files); err != nil {
		fmt.Printf("Warning: some units failed to stop: %v\n", err)
	}

	// Verify units are actually stopped
	if err := o.verifyUnitsStopped(state.Files); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	// Step 2: Remove quadlet files from target dir
	for _, f := range state.Files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to remove file %s: %v\n", f, err)
		}
	}

	// Step 3: Reload daemon so systemd forgets the removed units
	dbusMgr, err := deploy.NewSystemdManager()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	if err := dbusMgr.ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Step 4: Unregister project from state
	if err := stateMgr.UnregisterProject(o.projectName); err != nil {
		return fmt.Errorf("failed to unregister project: %w", err)
	}

	fmt.Println("Successfully removed project:", o.projectName)
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

func (o *Orchestrator) preprocess(composeFile string) ([]byte, error) {
	composeData, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

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
	podlet := transpile.NewPodletRunner(tempDir)
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
	cookerEngine := cooker.NewCooker(tempDir, targetDir, o.projectName, isRootless, portOffset)
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
	stateMgr, err := deploy.NewStateManager()
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
	dbusMgr, err := deploy.NewSystemdManager()
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
			fmt.Printf("Starting unit: %s\n", unitName)

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
		} else {
			fmt.Printf("Image already exists locally, skipping build: %s\n", imageTag)
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
				break
			}
		}
	}

	return nil
}

func (o *Orchestrator) stopUnits(projectFiles []string) error {
	dbusMgr, err := deploy.NewSystemdManager()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".container") {
			unitName := containerFileToUnitName(f)
			fmt.Printf("Stopping unit: %s\n", unitName)
			if err := dbusMgr.StopUnit(unitName); err != nil {
				return fmt.Errorf("failed to stop unit %s: %w", unitName, err)
			}
		}
	}

	return nil
}

func (o *Orchestrator) verifyUnitsStopped(projectFiles []string) error {
	dbusMgr, err := deploy.NewSystemdManager()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

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

func ContainerFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	return strings.TrimSuffix(base, ".container") + ".service"
}

func containerFileToUnitName(filePath string) string {
	return ContainerFileToUnitName(filePath)
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

func isRootless() bool {
	return os.Getuid() != 0
}
