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

		var containerUnits []string
		for _, f := range p.Files {
			if strings.HasSuffix(filepath.Base(f), ".container") {
				containerUnits = append(containerUnits, f)
			}
		}

		total := len(containerUnits)
		running := 0
		for _, f := range containerUnits {
			base := filepath.Base(f)
			short := shortName(base, prefix)
			unitName := prefix + short + ".service"
			st := unitMap[unitName]
			if st.sub == "running" || (st.active == "active" && st.sub != "failed") {
				running++
			}
		}

		status := "healthy"
		if total == 0 {
			status = "up"
		} else if running == 0 {
			status = "stopped"
		} else if running < total {
			status = "degraded"
		}

		rows = append(rows, projectRow{
			name:     p.ProjectName,
			status:   status,
			services: fmt.Sprintf("%d/%d", running, total),
			source:   p.SourcePath,
		})

		if output.IsJSONMode() {
			jp := output.ProjectJSON{
				Name:       p.ProjectName,
				SourcePath: p.SourcePath,
				Files:      len(p.Files),
				Status:     status,
				Services:   fmt.Sprintf("%d/%d", running, total),
			}
			if p.Resources != nil {
				jp.Resources = &output.ResourcesJSON{
					Containers: p.Resources.Containers,
					Networks:   p.Resources.Networks,
					Volumes:    p.Resources.Volumes,
					Images:     p.Resources.Images,
					Builds:     p.Resources.Builds,
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
	tw.AppendHeader(table.Row{"NAME", "STATUS", "SERVICES", "SOURCE"})
	for _, r := range rows {
		tw.AppendRow(table.Row{r.name, r.status, r.services, r.source})
	}
	logger.Print(tw.Render())

	return nil
}
