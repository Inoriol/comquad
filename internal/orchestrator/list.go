package orchestrator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/output"
)

type projectRow struct {
	name     string
	status   string
	services string
	jobs     string
	source   string
}

func (o *Orchestrator) List(filter string) error {
	stateMgr, err := o.newState()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	projects := stateMgr.ListProjects()

	if filter != "" {
		var filtered []deploy.ProjectState
		for _, p := range projects {
			if p.ProjectName == filter {
				filtered = append(filtered, p)
			}
		}
		projects = filtered
	}

	if len(projects) == 0 {
		if output.IsJSONMode() {
			return output.PrintJSON(&output.ProjectListData{Projects: []output.ProjectJSON{}})
		}
		logger.Print("No projects currently deployed.")
		return nil
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	allUnits, err := dbusMgr.ListAllUnits()
	if err != nil {
		return fmt.Errorf("failed to list units: %w", err)
	}

	unitMap := make(map[string]unitStatus)
	for _, u := range allUnits {
		unitMap[u.Name] = unitStatus{active: u.ActiveState, sub: u.SubState}
	}

	var rows []projectRow
	var jsonProjects []output.ProjectJSON

	for _, p := range projects {
		prefix := "cq-" + p.ProjectName + "-"

		var serviceContainerUnits []string
		var jobContainerUnits []string
		var timerUnits []string

		composeProject, _ := loadComposeProject(p.SourcePath)
		jobNames := make(map[string]bool)
		if composeProject != nil {
			for name := range composeProject.Jobs {
				jobNames[name] = true
			}
		}

		for _, f := range p.Files {
			if strings.HasSuffix(filepath.Base(f), ".container") {
				base := filepath.Base(f)
				containerName := strings.TrimSuffix(base, ".container")
				shortName := strings.TrimPrefix(containerName, prefix)
				if jobNames[shortName] {
					jobContainerUnits = append(jobContainerUnits, f)
				} else {
					serviceContainerUnits = append(serviceContainerUnits, f)
				}
			}
			if strings.HasSuffix(filepath.Base(f), ".timer") {
				timerUnits = append(timerUnits, f)
			}
		}

		total := len(serviceContainerUnits)
		running := 0
		for _, f := range serviceContainerUnits {
			base := filepath.Base(f)
			short := shortName(base, prefix)
			unitName := prefix + short + ".service"
			st := unitMap[unitName]
			if st.sub == "running" || (st.active == "active" && st.sub != "failed") {
				running++
			}
		}

		totalJobs := len(jobContainerUnits)
		activeTimers := 0
		for _, f := range timerUnits {
			unitName := TimerFileToUnitName(f)
			st := unitMap[unitName]
			if st.active == "active" {
				activeTimers++
			}
		}

		status := "healthy"
		if total == 0 && totalJobs == 0 {
			status = "up"
		} else if total > 0 && running == 0 {
			status = "stopped"
		} else if total > 0 && running < total {
			status = "degraded"
		}

		servicesStr := fmt.Sprintf("%d/%d", running, total)
		jobsStr := fmt.Sprintf("%d timers", activeTimers)
		if totalJobs > 0 {
			jobsStr = fmt.Sprintf("%d jobs, %d timers", totalJobs, activeTimers)
		}

		rows = append(rows, projectRow{
			name:     p.ProjectName,
			status:   status,
			services: servicesStr,
			jobs:     jobsStr,
			source:   p.SourcePath,
		})

		if output.IsJSONMode() {
			jp := output.ProjectJSON{
				Name:       p.ProjectName,
				SourcePath: p.SourcePath,
				Files:      len(p.Files),
				Status:     status,
				Services:   servicesStr,
				Jobs:       jobsStr,
			}
			if p.Resources != nil {
				jp.Resources = &output.ResourcesJSON{
					Containers: p.Resources.Containers,
					Networks:   p.Resources.Networks,
					Volumes:    p.Resources.Volumes,
					Images:     p.Resources.Images,
					Builds:     p.Resources.Builds,
					Secrets:    p.Resources.Secrets,
					Timers:     p.Resources.Timers,
				}
			}
			jsonProjects = append(jsonProjects, jp)
		}
	}

	if output.IsJSONMode() {
		return output.PrintJSON(&output.ProjectListData{Projects: jsonProjects})
	}

	tw := table.NewWriter()
	tw.SetStyle(table.StyleLight)
	tw.AppendHeader(table.Row{"NAME", "STATUS", "SERVICES", "JOBS", "SOURCE"})
	for _, r := range rows {
		tw.AppendRow(table.Row{r.name, r.status, r.services, r.jobs, r.source})
	}
	logger.Print(tw.Render())

	return nil
}
