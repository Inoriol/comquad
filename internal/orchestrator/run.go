package orchestrator

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/output"
)

func (o *Orchestrator) Run(runOpts RunOptions) error {
	if runOpts.JobName == "" {
		return fmt.Errorf("job name is required")
	}

	stateMgr, state, err := o.ensureProjectDeployed()
	if err != nil {
		return err
	}
	_ = stateMgr

	prefix := "cq-" + o.projectName + "-"
	jobContainerFile := prefix + runOpts.JobName + ".container"
	jobServiceName := prefix + runOpts.JobName + ".service"

	found := false
	for _, f := range state.Files {
		if filepath.Base(f) == jobContainerFile {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("job %q not found in project %s", runOpts.JobName, o.projectName)
	}

	composeProject, err := loadComposeProject(o.resolveComposeFile())
	if err != nil {
		return fmt.Errorf("failed to load compose project: %w", err)
	}

	job, exists := composeProject.Jobs[runOpts.JobName]
	if !exists {
		return fmt.Errorf("job %q not found in compose file", runOpts.JobName)
	}

	if job.Triggers != nil && job.Triggers.Manual != nil && !*job.Triggers.Manual {
		return fmt.Errorf("job %q has manual execution disabled", runOpts.JobName)
	}

	if runOpts.DryRun {
		if output.IsJSONMode() {
			return output.PrintJSON(&output.RunData{
				Project: o.projectName,
				Job:     runOpts.JobName,
				Action:  "run",
				DryRun:  true,
			})
		}
		logger.Print(fmt.Sprintf("Would run job: %s", runOpts.JobName))
		return nil
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to initialize systemd client: %w", err)
	}
	defer dbusMgr.Close()

	if !output.IsJSONMode() {
		logger.Action(fmt.Sprintf("Running job: %s", runOpts.JobName))
	}

	if err := dbusMgr.StartUnit(jobServiceName); err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	if runOpts.Follow {
		deployTime := time.Now().Format("2006-01-02 15:04:05")
		return o.FollowLogs(deployTime, runOpts.JobName, false)
	}

	if output.IsJSONMode() {
		return output.PrintJSON(&output.RunData{
			Project: o.projectName,
			Job:     runOpts.JobName,
			Action:  "run",
			Success: true,
		})
	}

	logger.Success(fmt.Sprintf("Job %s started", runOpts.JobName))
	return nil
}

func (o *Orchestrator) startTimers(projectFiles []string) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to initialize systemd client: %w", err)
	}
	defer dbusMgr.Close()

	var timerServices []string
	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".timer") {
			timerName := strings.TrimSuffix(f, ".timer")
			timerService := timerName + ".timer"
			timerServices = append(timerServices, timerService)
		}
	}

	for _, timerService := range timerServices {
		if err := dbusMgr.StartUnit(timerService); err != nil {
			logger.Warn(fmt.Sprintf("failed to start timer %s: %v", timerService, err))
		}
	}

	return nil
}

func (o *Orchestrator) stopTimers(projectFiles []string) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to initialize systemd client: %w", err)
	}
	defer dbusMgr.Close()

	var timerServices []string
	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".timer") {
			timerName := strings.TrimSuffix(f, ".timer")
			timerService := timerName + ".timer"
			timerServices = append(timerServices, timerService)
		}
	}

	for _, timerService := range timerServices {
		if err := dbusMgr.StopUnit(timerService); err != nil {
			logger.Warn(fmt.Sprintf("failed to stop timer %s: %v", timerService, err))
		}
	}

	return nil
}

func (o *Orchestrator) enableTimers(projectFiles []string) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to initialize systemd client: %w", err)
	}
	defer dbusMgr.Close()

	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".timer") {
			timerName := strings.TrimSuffix(f, ".timer")
			timerService := timerName + ".timer"
			if err := dbusMgr.EnableUnit(timerService); err != nil {
				logger.Warn(fmt.Sprintf("failed to enable timer %s: %v", timerService, err))
			}
		}
	}

	return nil
}

func (o *Orchestrator) disableTimers(projectFiles []string) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to initialize systemd client: %w", err)
	}
	defer dbusMgr.Close()

	for _, f := range projectFiles {
		if strings.HasSuffix(f, ".timer") {
			timerName := strings.TrimSuffix(f, ".timer")
			timerService := timerName + ".timer"
			if err := dbusMgr.DisableUnit(timerService); err != nil {
				logger.Warn(fmt.Sprintf("failed to disable timer %s: %v", timerService, err))
			}
		}
	}

	return nil
}

func TimerFileToUnitName(filePath string) string {
	base := filePath
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	return strings.TrimSuffix(base, ".timer") + ".timer"
}

func (o *Orchestrator) collectTimerFiles(targetDir string) ([]string, error) {
	stateMgr, state, err := o.ensureProjectDeployed()
	if err != nil {
		return nil, err
	}
	_ = stateMgr

	var timerFiles []string
	for _, f := range state.Files {
		if strings.HasSuffix(f, ".timer") {
			timerFiles = append(timerFiles, targetDir+"/"+f)
		}
	}
	return timerFiles, nil
}

func (o *Orchestrator) getJobNames() []string {
	composeProject, err := loadComposeProject(o.resolveComposeFile())
	if err != nil {
		return nil
	}

	var jobNames []string
	for name := range composeProject.Jobs {
		jobNames = append(jobNames, name)
	}
	return jobNames
}

func (o *Orchestrator) isJob(name string) bool {
	composeProject, err := loadComposeProject(o.resolveComposeFile())
	if err != nil {
		return false
	}
	_, exists := composeProject.Jobs[name]
	return exists
}

func (o *Orchestrator) getJobContainerFiles() []string {
	stateMgr, state, err := o.ensureProjectDeployed()
	if err != nil {
		return nil
	}
	_ = stateMgr

	composeProject, err := loadComposeProject(o.resolveComposeFile())
	if err != nil {
		return nil
	}

	var jobFiles []string
	prefix := "cq-" + o.projectName + "-"
	for jobName := range composeProject.Jobs {
		containerFile := prefix + jobName + ".container"
		for _, f := range state.Files {
			if f == containerFile {
				jobFiles = append(jobFiles, f)
				break
			}
		}
	}
	return jobFiles
}

func (o *Orchestrator) separateJobAndServiceFiles(projectFiles []string) (serviceFiles []string, jobFiles []string) {
	composeProject, err := loadComposeProject(o.resolveComposeFile())
	if err != nil {
		return projectFiles, nil
	}

	prefix := "cq-" + o.projectName + "-"
	jobNames := make(map[string]bool)
	for name := range composeProject.Jobs {
		jobNames[prefix+name+".container"] = true
	}

	for _, f := range projectFiles {
		if jobNames[f] {
			jobFiles = append(jobFiles, f)
		} else {
			serviceFiles = append(serviceFiles, f)
		}
	}
	return serviceFiles, jobFiles
}
