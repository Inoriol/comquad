package orchestrator

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/reconcile"
)

const (
	defaultPortOffset = 2000
	startUnitWaitTime = 10 * time.Second
)

type Orchestrator struct {
	projectName string
	cwd         string

	newState       func() (deploy.StateStore, error)
	newSystemd     func() (deploy.SystemdClient, error)
	listContainers func(projectName string, all bool) ([]ContainerInfo, error)
	newJournalCmd  func(name string, args ...string) *exec.Cmd
}

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

func (o *Orchestrator) ProjectName() string {
	return o.projectName
}

func (o *Orchestrator) Up(pullStrategy string, follow bool, dryRun bool, noDiff bool) error {
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

	podmanVersion, versionErr := deploy.DetectPodmanVersion()
	if versionErr != nil {
		if !dryRun {
			logger.Warn("could not detect podman version, assuming latest: " + versionErr.Error())
		}
	}

	targetDir, err := o.resolveTargetDir()
	if err != nil {
		return err
	}

	isRootless := deploy.IsRootless()
	selinuxEnabled := deploy.IsSELinuxEnabled()

	portOffset := 0
	if isRootless {
		portOffset = defaultPortOffset
		if envOffset := os.Getenv("ROOTLESS_PORT_OFFSET"); envOffset != "" {
			if parsed, err := strconv.Atoi(envOffset); err == nil && parsed > 0 {
				portOffset = parsed
			}
		}
	}

	if selinuxEnabled {
		logger.Action(fmt.Sprintf("SELinux detected, adding :z labels to all volumes"))
	}

	secretsDir, err := resolveSecretsDir(o.projectName)
	if err != nil {
		return fmt.Errorf("failed to resolve secrets directory: %w", err)
	}

	buildCacheDir, err := resolveBuildCacheDir(o.projectName)
	if err != nil {
		return fmt.Errorf("failed to resolve build cache directory: %w", err)
	}

	opts := []c2q.TranspileOption{
		c2q.WithProjectName(o.projectName),
		c2q.WithLabels(map[string]string{
			"com.comquad.managed": "true",
			"com.comquad.project": o.projectName,
		}),
		c2q.WithAutoUpdate(),
		c2q.WithSecretsDirectory(secretsDir),
		c2q.WithBuildCacheDir(buildCacheDir),
		c2q.WithDockerfileNormalization(),
	}

	if versionErr == nil {
		opts = append(opts, c2q.WithPodmanVersion(podmanVersion))
	}

	if portOffset > 0 {
		opts = append(opts, c2q.WithPortOffset(portOffset))
		logger.Action(fmt.Sprintf("Applied port offset %d for rootless mode", portOffset))
	}
	opts = append(opts, c2q.WithInfo(logger.Action))
	if !selinuxEnabled {
		opts = append(opts, c2q.WithoutSELinux())
	}
	if dryRun {
		opts = append(opts, c2q.WithDryRun())
	}

	logger.Action("Transpiling compose configuration...")
	units, err := c2q.TranspileFile(composeFile, opts...)
	if err != nil {
		return fmt.Errorf("transpilation failed: %w", err)
	}

	stripServiceName(units)

	baselineDir, err := resolveBaselineDir(o.projectName)
	if err != nil {
		return fmt.Errorf("failed to resolve baseline directory: %w", err)
	}

	prefix := "cq-" + o.projectName + "-"
	plan, err := reconcile.Compute(targetDir, baselineDir, prefix, units)
	if err != nil {
		return fmt.Errorf("computing changes: %w", err)
	}

	if dryRun {
		return o.printDryRun(units, targetDir, pullStrategy, plan)
	}

	if !noDiff && o.projectDeployed() && plan.HasChanges() {
		fmt.Print(colorizeDiff(plan.Diff()))
		if isTerminal(os.Stdin) {
			proceed, err := confirmUpdate()
			if err != nil {
				return err
			}
			if !proceed {
				logger.Print("Update cancelled — no changes applied.")
				return nil
			}
		}
	}

	logger.Action("Reconciling quadlet files...")
	result, err := reconcile.Apply(targetDir, baselineDir, plan)
	if err != nil {
		return fmt.Errorf("reconciling quadlet files: %w", err)
	}
	o.reportReconcile(result)

	projectFiles, err := o.collectProjectFiles(targetDir)
	if err != nil {
		return err
	}

	priorState, hadPriorState := o.getProjectState()

	if err := o.registerState(projectFiles); err != nil {
		o.rollbackDeploy(plan, priorState, hadPriorState)
		return err
	}

	logger.Action("Handling images...")
	if err := o.handleImages(projectFiles, units, pullStrategy); err != nil {
		o.rollbackDeploy(plan, priorState, hadPriorState)
		return err
	}

	deployTime := time.Now().Format("2006-01-02 15:04:05")

	if follow {
		logger.Print("Following logs for project: " + o.projectName)
		logErrCh := make(chan error, 1)
		go func() {
			logErrCh <- o.FollowLogs(deployTime, "", false)
		}()

		time.Sleep(500 * time.Millisecond)

		logger.Action("Starting services...")
		if err := o.startUnits(projectFiles, result); err != nil {
			o.rollbackDeploy(plan, priorState, hadPriorState)
			return fmt.Errorf("failed to start services: %w", err)
		}

		logger.Success("Successfully deployed project: " + o.projectName)
		return <-logErrCh
	}

	logger.Action("Starting services...")
	if err := o.startUnits(projectFiles, result); err != nil {
		o.rollbackDeploy(plan, priorState, hadPriorState)
		return fmt.Errorf("failed to start services: %w", err)
	}

	logger.Success("Successfully deployed project: " + o.projectName)
	return nil
}

func (o *Orchestrator) getProjectState() (deploy.ProjectState, bool) {
	sm, err := o.newState()
	if err != nil {
		return deploy.ProjectState{}, false
	}
	return sm.GetProject(o.projectName)
}

// rollbackDeploy reverts the quadlet files and baseline touched by a reconcile
// back to their pre-deploy state, so a failed deploy leaves the previous
// deployment intact instead of tearing it down.
func (o *Orchestrator) rollbackDeploy(plan reconcile.Plan, priorState deploy.ProjectState, hadPriorState bool) {
	for _, fp := range plan.Files {
		if fp.Status == reconcile.StatusUnchanged {
			continue
		}
		switch fp.Status {
		case reconcile.StatusCreated:
			os.Remove(fp.TargetPath)
		case reconcile.StatusChanged:
			os.WriteFile(fp.TargetPath, []byte(fp.OldContent), 0644)
		case reconcile.StatusRemoved:
			if fp.OldContent != "" {
				os.WriteFile(fp.TargetPath, []byte(fp.OldContent), 0644)
			}
		}
		if fp.OldBaseline != "" {
			os.WriteFile(fp.BasePath, []byte(fp.OldBaseline), 0644)
		} else {
			os.Remove(fp.BasePath)
		}
	}
	if sm, err := o.newState(); err == nil {
		if hadPriorState {
			sm.RegisterProject(priorState)
		} else {
			sm.UnregisterProject(o.projectName)
		}
	}
	if dbusMgr, err := o.newSystemd(); err == nil {
		defer dbusMgr.Close()
		dbusMgr.ReloadDaemon()
	}
}

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

func (o *Orchestrator) projectDeployed() bool {
	stateMgr, err := o.newState()
	if err != nil {
		return false
	}
	_, exists := stateMgr.GetProject(o.projectName)
	return exists
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func noColorDisabled() bool {
	_, found := os.LookupEnv("NO_COLOR")
	return found
}

func colorizeDiff(diff string) string {
	if !isTerminal(os.Stdout) || noColorDisabled() {
		return diff
	}

	const (
		reset = "\033[0m"
		red   = "\033[31m"
		green = "\033[32m"
		cyan  = "\033[36m"
		bold  = "\033[1m"
	)

	lines := strings.Split(diff, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			b.WriteString(bold + line + reset)
		case strings.HasPrefix(line, "@@"):
			b.WriteString(cyan + line + reset)
		case strings.HasPrefix(line, "+"):
			b.WriteString(green + line + reset)
		case strings.HasPrefix(line, "-"):
			b.WriteString(red + line + reset)
		default:
			b.WriteString(line)
		}
	}
	return b.String()
}

func confirmUpdate() (bool, error) {
	fmt.Print("Apply changes? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

func ContainerFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	return strings.TrimSuffix(base, ".container") + ".service"
}

func NetworkFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, ".network")
	return nameWithoutExt + "-network.service"
}

func VolumeFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, ".volume")
	return nameWithoutExt + "-volume.service"
}

func ImageFileToUnitName(filePath string) string {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, ".image")
	return nameWithoutExt + "-image.service"
}

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

func resolveBaselineDir(projectName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "comquad", "baseline", projectName), nil
}

func resolveBuildCacheDir(projectName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "comquad", "builds", projectName), nil
}
