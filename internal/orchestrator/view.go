package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/Inoriol/comquad/internal/deploy"
	"github.com/Inoriol/comquad/internal/logger"
)

type unitStatus struct {
	active string
	sub    string
}

type serviceRow struct {
	name   string
	status string
	image  string
	nets   []string
	vols   []string
}

type resourceRow struct {
	name string
	kind string
	info string
}

func shortName(fileName, projectPrefix string) string {
	name := strings.TrimPrefix(fileName, projectPrefix)
	for _, ext := range []string{".container", ".image", ".network", ".volume", ".build"} {
		name = strings.TrimSuffix(name, ext)
	}
	return name
}

func resolveImageFromQuadlet(files map[string]string, ref string) string {
	for path, content := range files {
		if filepath.Base(path) == ref {
			for _, line := range strings.Split(content, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "Image=") {
					return strings.TrimPrefix(trimmed, "Image=")
				}
			}
		}
	}
	return ref
}

func displayImage(containerImage string, imageFiles map[string]string) string {
	if strings.HasSuffix(containerImage, ".image") {
		resolved := resolveImageFromQuadlet(imageFiles, containerImage)
		return displayImage(resolved, nil)
	}
	return shortenImage(containerImage)
}

func shortenImage(img string) string {
	img = strings.TrimPrefix(img, "docker.io/library/")
	img = strings.TrimPrefix(img, "docker.io/")
	return img
}

// View shows systemd units for a project or the contents of a specific unit file.
func (o *Orchestrator) View(projectArg string) error {
	_, state, err := o.ensureProjectDeployed()
	if err != nil {
		return err
	}

	if projectArg == "" {
		return o.viewProject(state)
	}

	return o.viewUnit(state, projectArg)
}

func (o *Orchestrator) viewProject(state deploy.ProjectState) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	allUnits, err := dbusMgr.ListAllUnits()
	if err != nil {
		return fmt.Errorf("failed to list units: %w", err)
	}

	prefix := "cq-" + o.projectName + "-"
	unitMap := make(map[string]unitStatus)
	for _, u := range allUnits {
		if strings.HasPrefix(u.Name, "cq-"+o.projectName) {
			unitMap[u.Name] = unitStatus{active: u.ActiveState, sub: u.SubState}
		}
	}

	if len(unitMap) == 0 {
		logger.Printf("No units found for project %s\n", o.projectName)
		return nil
	}

	fileContents := make(map[string]string)
	for _, f := range state.Files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		fileContents[f] = string(data)
	}

	var services []serviceRow
	var resources []resourceRow

	for _, f := range state.Files {
		base := filepath.Base(f)
		short := shortName(base, prefix)

		if strings.HasSuffix(base, ".container") {
			unitName := prefix + short + ".service"
			st := unitMap[unitName]
			status := st.active
			if status == "" {
				status = "unknown"
			}
			if st.sub == "running" {
				status = "running"
			} else if st.active == "active" {
				status = "up"
			} else if st.active == "inactive" {
				status = "stopped"
			} else if st.active == "failed" {
				status = "failed"
			}

			row := serviceRow{name: short, status: status}

			content := fileContents[f]
			for _, line := range strings.Split(content, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "Image=") {
					row.image = displayImage(strings.TrimPrefix(trimmed, "Image="), fileContents)
				}
				if strings.HasPrefix(trimmed, "Network=") {
					netRef := strings.TrimPrefix(trimmed, "Network=")
					netRef = strings.TrimSuffix(netRef, ".network")
					netShort := strings.TrimPrefix(netRef, prefix)
					row.nets = append(row.nets, netShort)
				}
				if strings.HasPrefix(trimmed, "Volume=") {
					volRef := strings.TrimPrefix(trimmed, "Volume=")
					parts := strings.SplitN(volRef, ":", 3)
					volName := strings.TrimSuffix(parts[0], ".volume")
					volShort := strings.TrimPrefix(volName, prefix)
					row.vols = append(row.vols, volShort)
				}
			}
			services = append(services, row)
		}

		if strings.HasSuffix(base, ".image") {
			img := ""
			for _, line := range strings.Split(fileContents[f], "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "Image=") {
					img = shortenImage(strings.TrimPrefix(strings.TrimSpace(line), "Image="))
					break
				}
			}
			resources = append(resources, resourceRow{name: short + ".image", kind: "image", info: img})
		}

		if strings.HasSuffix(base, ".network") {
			resources = append(resources, resourceRow{name: short + ".network", kind: "network", info: "—"})
		}

		if strings.HasSuffix(base, ".volume") {
			resources = append(resources, resourceRow{name: short + ".volume", kind: "volume", info: "—"})
		}
	}

	sort.Slice(services, func(i, j int) bool { return services[i].name < services[j].name })
	sort.Slice(resources, func(i, j int) bool { return resources[i].name < resources[j].name })

	activeCount := 0
	for _, s := range services {
		if s.status == "running" || s.status == "up" {
			activeCount++
		}
	}

	status := "healthy"
	if len(services) == 0 {
		status = "up"
	} else if activeCount == 0 {
		status = "stopped"
	} else if activeCount < len(services) {
		status = "degraded"
	}

	logger.Printf("Project:  %s\n", o.projectName)
	logger.Printf("Source:   %s\n", state.SourcePath)
	logger.Printf("Status:   %s\n\n", status)

	if len(services) > 0 {
		logger.Print("SERVICES")
		tw := table.NewWriter()
		tw.SetStyle(table.StyleLight)
		tw.AppendHeader(table.Row{"NAME", "STATUS", "IMAGE", "NETWORKS", "VOLUMES"})
		for _, s := range services {
			netStr := strings.Join(s.nets, ", ")
			if netStr == "" {
				netStr = "—"
			}
			volStr := strings.Join(s.vols, ", ")
			if volStr == "" {
				volStr = "—"
			}
			tw.AppendRow(table.Row{s.name, s.status, s.image, netStr, volStr})
		}
		logger.Print(tw.Render())
	}

	if len(resources) > 0 {
		logger.Print("RESOURCES")
		tw := table.NewWriter()
		tw.SetStyle(table.StyleLight)
		tw.AppendHeader(table.Row{"NAME", "TYPE", "INFO"})
		for _, r := range resources {
			tw.AppendRow(table.Row{r.name, r.kind, r.info})
		}
		logger.Print(tw.Render())
	}

	return nil
}

func (o *Orchestrator) viewUnit(state deploy.ProjectState, arg string) error {
	if found := o.matchContainer(state, arg); found != "" {
		return o.printFile(found)
	}

	if found := o.matchQuadletResource(state, arg); found != "" {
		return o.printFile(found)
	}

	var suggestions []string
	for _, f := range state.Files {
		suggestions = append(suggestions, filepath.Base(f))
	}
	return fmt.Errorf("no unit found for '%s', did you mean: %s", arg, strings.Join(suggestions, ", "))
}

func (o *Orchestrator) matchContainer(state deploy.ProjectState, arg string) string {
	return MatchFirstContainer(o.projectName, state, arg)
}

func (o *Orchestrator) matchQuadletResource(state deploy.ProjectState, arg string) string {
	return MatchQuadletResource(o.projectName, state, arg)
}

func (o *Orchestrator) printFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	logger.Printf("── %s ──\n", filepath.Base(path))
	logger.Printf("%s", string(content))
	if !strings.HasSuffix(string(content), "\n") {
		logger.Print("")
	}

	return nil
}
