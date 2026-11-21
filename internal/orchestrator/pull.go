package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
	"github.com/Inoriol/comquad/compose2quadlet/mapper"
	"github.com/Inoriol/comquad/compose2quadlet/pseudo"
	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/output"
	"github.com/Inoriol/comquad/internal/reconcile"
)

func (o *Orchestrator) Pull(pullOpts PullOptions) error {
	if !output.IsJSONMode() {
		logger.Action("Reading compose file...")
	}
	composeFile := o.resolveComposeFile()
	if composeFile == "" {
		return fmt.Errorf("no compose file found in current directory (looked for compose.yaml, compose.yml, docker-compose.yaml, docker-compose.yml, podman-compose.yaml, podman-compose.yml)")
	}

	if !pullOpts.DryRun && !deploy.StateFileExists() {
		if !output.IsJSONMode() {
			logger.Action("First deployment detected — checking prerequisites...")
		}
		if err := deploy.ValidatePodmanVersion(); err != nil {
			return fmt.Errorf("prerequisite check failed: %w", err)
		}
	}

	podmanVersion, versionErr := deploy.DetectPodmanVersion()
	if versionErr != nil {
		if !pullOpts.DryRun && !output.IsJSONMode() {
			logger.Warn("could not detect podman version, assuming latest: " + versionErr.Error())
		}
	}

	targetDir, err := o.resolveTargetDir()
	if err != nil {
		return err
	}

	systemdUnitDir, err := o.resolveSystemdUnitDir()
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

	if selinuxEnabled && !output.IsJSONMode() {
		logger.Action(fmt.Sprintf("SELinux detected, adding :z/:Z labels to volumes"))
	}

	secretsDir, err := resolveSecretsDir(o.projectName)
	if err != nil {
		return fmt.Errorf("failed to resolve secrets directory: %w", err)
	}

	buildCacheDir, err := resolveBuildCacheDir(o.projectName)
	if err != nil {
		return fmt.Errorf("failed to resolve build cache directory: %w", err)
	}

	autoNSEnabled := os.Getenv("COMQUAD_AUTONS") == "1"
	var autoNSRange *deploy.AutoNSRange
	if autoNSEnabled {
		if isRootless {
			return fmt.Errorf("COMQUAD_AUTONS=1 is only supported in rootful mode")
		}
		if versionErr == nil && !podmanVersion.AtLeast(4, 4) {
			return fmt.Errorf("AutoNS requires podman >= 4.4.0 for idmap volume support")
		}
		if err := deploy.ValidateContainersEntry(); err != nil {
			return err
		}

		subuidEntries, err := deploy.ParseSubuid()
		if err != nil {
			return fmt.Errorf("failed to parse /etc/subuid: %w", err)
		}

		stateMgr, stateErr := o.newState()
		var usedBases []int
		if stateErr == nil {
			usedBases = deploy.CollectUsedBases(stateMgr.ListProjects())
		}
		stackBase := deploy.AllocateStackRange(subuidEntries, usedBases)

		composeProject, err := loadComposeProject(composeFile)
		if err != nil {
			return fmt.Errorf("failed to load compose project for AutoNS anchor selection: %w", err)
		}
		anchor := mapper.SelectAnchor(composeProject.Services)
		if anchor == "" {
			return fmt.Errorf("AutoNS: no services found in compose file for anchor selection")
		}

		autoNSRange = &deploy.AutoNSRange{
			BaseUID: stackBase,
			Size:    deploy.AutoNSRangeSize,
			Anchor:  anchor,
		}

		if !output.IsJSONMode() {
			logger.Action(fmt.Sprintf("AutoNS enabled: anchor=%s, uid_base=%d", anchor, stackBase))
		}
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
		c2q.WithoutServiceName(),
	}

	if autoNSEnabled {
		opts = append(opts, c2q.WithAutoNS(autoNSRange.BaseUID, autoNSRange.Anchor))
	}

	if versionErr == nil {
		opts = append(opts, c2q.WithPodmanVersion(podmanVersion))
	}

	if portOffset > 0 {
		opts = append(opts, c2q.WithPortOffset(portOffset))
		if !output.IsJSONMode() {
			logger.Action(fmt.Sprintf("Applied port offset %d for rootless mode", portOffset))
		}
	}
	if !output.IsJSONMode() {
		opts = append(opts, c2q.WithInfo(logger.Action))
	}
	if !selinuxEnabled {
		opts = append(opts, c2q.WithoutSELinux())
	}
	if pullOpts.DryRun {
		opts = append(opts, c2q.WithDryRun())
	}
	if os.Getenv("COMQUAD_SKIP_REGISTRY_NORMALIZATION") != "" {
		opts = append(opts, c2q.WithoutImageNormalization())
	}
	if os.Getenv("COMQUAD_SYSTEMD_SPECIFIERS") != "" {
		opts = append(opts, c2q.WithSystemdSpecifiers())
	}

	if !output.IsJSONMode() {
		logger.Action("Transpiling compose configuration...")
	}
	units, err := c2q.TranspileFile(composeFile, opts...)
	if err != nil {
		return fmt.Errorf("transpilation failed: %w", err)
	}

	runtimeSecretsDir, err := deploy.ResolveRuntimeSecretsDir()
	if err != nil {
		return fmt.Errorf("failed to resolve runtime secrets directory: %w", err)
	}
	units = pseudo.ConvertPseudoUnits(units, runtimeSecretsDir)

	pullStrategy := pullOpts.PullStrategy
	if pullStrategy == "" {
		pullStrategy = "always"
	}

	strat, err := ParsePullStrategy(pullStrategy)
	if err != nil {
		return err
	}
	setPolicyOnImageUnits(units, strat)
	if versionErr == nil && !podmanVersion.AtLeast(5, 6) {
		if !output.IsJSONMode() {
			logger.Info("Policy= in .image units requires podman >= 5.6.0 — pull strategy may not be enforced")
		}
	}

	baselineDir, err := resolveBaselineDir(o.projectName)
	if err != nil {
		return fmt.Errorf("failed to resolve baseline directory: %w", err)
	}

	prefix := "cq-" + o.projectName + "-"
	plan, err := reconcile.Compute(targetDir, systemdUnitDir, baselineDir, prefix, units)
	if err != nil {
		return fmt.Errorf("computing changes: %w", err)
	}

	if pullOpts.DryRun {
		return o.printDryRun(units, targetDir, pullStrategy, plan)
	}

	if !output.IsJSONMode() {
		logger.Action("Reconciling quadlet files...")
	}
	result, err := reconcile.Apply(targetDir, systemdUnitDir, baselineDir, plan)
	if err != nil {
		return fmt.Errorf("reconciling quadlet files: %w", err)
	}
	o.reportReconcile(result)

	projectFiles, err := o.collectProjectFiles(targetDir, systemdUnitDir)
	if err != nil {
		return err
	}

	if err := o.copyPseudoQuadlets(projectFiles); err != nil {
		return err
	}

	priorState, hadPriorState := o.getProjectState()

	if err := o.registerState(projectFiles, autoNSRange); err != nil {
		o.rollbackDeploy(plan, priorState, hadPriorState)
		return err
	}

	if !output.IsJSONMode() {
		logger.Action("Preparing units...")
	}
	if err := o.prepareUnits(projectFiles, result); err != nil {
		o.rollbackDeploy(plan, priorState, hadPriorState)
		return err
	}

	if !output.IsJSONMode() {
		logger.Action("Pulling images...")
	}
	if err := o.handleImages(projectFiles, units, pullStrategy); err != nil {
		o.rollbackDeploy(plan, priorState, hadPriorState)
		return err
	}

	var imagesPulled []string
	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".image") {
			name := strings.TrimSuffix(filepath.Base(f), ".image")
			name = strings.TrimPrefix(name, prefix)
			imagesPulled = append(imagesPulled, name)
		}
	}

	if output.IsJSONMode() {
		return output.PrintJSON(&output.PullData{
			Project:      o.projectName,
			ImagesPulled: imagesPulled,
			Success:      true,
		})
	}

	logger.Success("Successfully pulled images for project: " + o.projectName)
	return nil
}
