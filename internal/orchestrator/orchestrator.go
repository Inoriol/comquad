package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/preprocess"
)

const (
	defaultPortOffset = 2000
	startUnitWaitTime = 10 * time.Second
)

// Orchestrator wires all internal packages together and drives
// the lifecycle of a comquad project.
type Orchestrator struct {
	projectName string
	cwd         string

	newState       func() (deploy.StateStore, error)
	newSystemd     func() (deploy.SystemdClient, error)
	listContainers func(projectName string, all bool) ([]ContainerInfo, error)
	newJournalCmd  func(name string, args ...string) *exec.Cmd
}

// NewOrchestrator creates a new Orchestrator for the current working directory.
func NewOrchestrator(projectName string) (*Orchestrator, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	if projectName == "" {
		projectName = filepath.Base(cwd)
	}

	if err := validateProjectName(projectName); err != nil {
		return nil, err
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
		newJournalCmd: func(name string, args ...string) *exec.Cmd {
			return exec.Command(name, args...)
		},
	}, nil
}

// ProjectName returns the resolved project name.
func (o *Orchestrator) ProjectName() string {
	return o.projectName
}

// Up preprocesses, transpiles, cooks and deploys the project
// defined in the compose.yaml in the current working directory.
func (o *Orchestrator) Up(pullStrategy string, follow bool, dryRun bool) error {
	logger.Action("Reading compose file...")
	composeFile := findComposeFile(o.cwd)
	if composeFile == "" {
		return fmt.Errorf("no compose file found in current directory (looked for compose.yaml, compose.yml, docker-compose.yaml, docker-compose.yml)")
	}

	if !dryRun && !deploy.StateFileExists() {
		logger.Action("First deployment detected — checking prerequisites...")
		if err := deploy.ValidatePodmanVersion(); err != nil {
			return fmt.Errorf("prerequisite check failed: %w", err)
		}
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

	composeData, err := os.ReadFile(composeFile)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}

	serviceSpecs, err := preprocess.ExtractServiceImageSpecs(composeData)
	if err != nil {
		return fmt.Errorf("failed to extract service image specs: %w", err)
	}

	secretDefs, serviceSecretRefs, err := preprocess.ExtractSecretSpecs(composeData, o.cwd)
	if err != nil {
		return fmt.Errorf("failed to extract secret specs: %w", err)
	}

	secretsDir, err := resolveSecretsDir(o.projectName)
	if err != nil {
		return fmt.Errorf("failed to resolve secrets directory: %w", err)
	}

	logger.Action("Preprocessing compose configuration...")
	processedYaml, err := o.preprocess(composeData)
	if err != nil {
		return err
	}

	logger.Action("Transpiling to quadlet files...")
	if err := o.transpile(processedYaml, tempDir); err != nil {
		return err
	}

	isRootless := deploy.IsRootless()

	if dryRun {
		previewDir, err := os.MkdirTemp("", "comquad-preview-*")
		if err != nil {
			return fmt.Errorf("failed to create preview directory: %w", err)
		}
		defer os.RemoveAll(previewDir)

		logger.Action("Generating quadlet files (dry run)...")
		previewContents, err := o.cook(tempDir, previewDir, isRootless)
		if err != nil {
			return err
		}

		previewContents = o.graft(previewContents, serviceSpecs, secretDefs, serviceSecretRefs, secretsDir, true)

		projectFiles, err := o.collectProjectFiles(previewDir)
		if err != nil {
			return err
		}

		return o.printDryRun(projectFiles, previewContents, previewDir, targetDir, pullStrategy)
	}

	logger.Action("Generating quadlet files...")
	fileContents, err := o.cook(tempDir, targetDir, isRootless)
	if err != nil {
		return err
	}

	fileContents = o.graft(fileContents, serviceSpecs, secretDefs, serviceSecretRefs, secretsDir, false)

	projectFiles, err := o.collectProjectFiles(targetDir)
	if err != nil {
		return err
	}

	cleanup := func() {
		for _, f := range projectFiles {
			os.Remove(f)
		}
		if sm, err := o.newState(); err == nil {
			sm.UnregisterProject(o.projectName)
		}
		os.RemoveAll(secretsDir)
		if dbusMgr, err := o.newSystemd(); err == nil {
			defer dbusMgr.Close()
			dbusMgr.ReloadDaemon(projectFiles...)
		}
	}

	if err := o.registerState(projectFiles); err != nil {
		cleanup()
		return err
	}

	logger.Action("Handling images...")
	if err := o.handleImages(projectFiles, fileContents, pullStrategy); err != nil {
		cleanup()
		return err
	}

	deployTime := time.Now().Format("2006-01-02 15:04:05")

	logger.Action("Starting services...")
	if err := o.startUnits(projectFiles); err != nil {
		cleanup()
		return fmt.Errorf("failed to start services: %w", err)
	}

	logger.Success("Successfully deployed project: " + o.projectName)

	if follow {
		logger.Print("Following logs for project: " + o.projectName)
		return o.FollowLogs(deployTime, "", false)
	}

	return nil
}

// ensureProjectDeployed returns the state store and project state, or an error
// if the project is not deployed.
func (o *Orchestrator) ensureProjectDeployed() (deploy.StateStore, deploy.ProjectState, error) {
	stateMgr, err := o.newState()
	if err != nil {
		return nil, deploy.ProjectState{}, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	state, exists := stateMgr.GetProject(o.projectName)
	if !exists {
		return nil, deploy.ProjectState{}, fmt.Errorf("project %s is not deployed", o.projectName)
	}

	return stateMgr, state, nil
}

// ContainerFileToUnitName derives the systemd unit name from a quadlet file path.
// e.g. /path/to/cq-myapp-web.container -> cq-myapp-web.service
func ContainerFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	return strings.TrimSuffix(base, ".container") + ".service"
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

// ImageFileToUnitName derives the systemd unit name from an image quadlet file path.
// e.g. /path/to/cq-myapp-app.image -> cq-myapp-app-image.service
func ImageFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, ".image")
	return nameWithoutExt + "-image.service"
}

// BuildFileToUnitName derives the systemd unit name from a build quadlet file path.
// e.g. /path/to/cq-myapp-app.build -> cq-myapp-app-build.service
func BuildFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, ".build")
	return nameWithoutExt + "-build.service"
}

func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("project name cannot start with '.'")
	}
	if name == ".." {
		return fmt.Errorf("invalid project name: '..'")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return fmt.Errorf("project name %q contains invalid character: %q (only letters, digits, hyphens, and underscores are allowed)", name, r)
		}
	}
	return nil
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
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

func resolveSecretsDir(projectName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "comquad", "secrets", projectName), nil
}
