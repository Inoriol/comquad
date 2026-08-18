package orchestrator

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/Inoriol/comquad/internal/logger"
)

// Ps shows the current state of containers for the project.
func (o *Orchestrator) Ps(all bool) error {
	if _, _, err := o.ensureProjectDeployed(); err != nil {
		return err
	}

	containers, err := o.listContainers(o.projectName, all)
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		logger.Printf("No containers found for project %s\n", o.projectName)
		return nil
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	var unitNames []string
	for _, c := range containers {
		unitNames = append(unitNames, "cq-"+c.Name+".service")
	}

	if len(unitNames) > 0 {
		units, err := dbusMgr.ListUnitsByNames(unitNames)
		if err != nil {
			return fmt.Errorf("failed to list units: %w", err)
		}

		unitStateMap := make(map[string]dbus.UnitStatus)
		for _, u := range units {
			unitStateMap[u.Name] = u
		}

		for i := range containers {
			unitName := "cq-" + containers[i].Name + ".service"
			if u, ok := unitStateMap[unitName]; ok {
				containers[i].DBusActive = u.ActiveState
				containers[i].DBusSub = u.SubState
			}
		}
	}

	sort.SliceStable(containers, func(i, j int) bool {
		if containers[i].State == "running" && containers[j].State != "running" {
			return true
		}
		if containers[i].State != "running" && containers[j].State == "running" {
			return false
		}
		if containers[i].State == "exited" && containers[j].State != "exited" && containers[j].State != "running" {
			return true
		}
		if containers[i].State != "exited" && containers[i].State != "running" && containers[j].State == "exited" {
			return false
		}
		return strings.ToLower(containers[i].Name) < strings.ToLower(containers[j].Name)
	})

	printPsTable(containers)

	return nil
}

func printPsTable(containers []ContainerInfo) {
	tw := table.NewWriter()
	tw.SetStyle(table.StyleLight)
	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 3, WidthMax: 30},
	})
	tw.AppendHeader(table.Row{"NAME", "IMAGE", "COMMAND", "SERVICE", "CREATED", "STATUS", "PORTS"})

	cmdMaxLen := 30 * 2 // 2 lines at WidthMax

	for _, c := range containers {
		status := c.Status
		if c.State == "exited" && c.ExitCode != 0 {
			status = fmt.Sprintf("Exited (%d) %s", c.ExitCode, c.Status)
		} else if c.State == "dead" {
			status = "Dead"
		}

		created := formatCreated(c.CreatedAt, c.ExitedAt, c.State)
		portStr := formatPorts(c.Ports, c.ExposedPorts)

		command := c.Command
		if len(command) > cmdMaxLen {
			command = command[:cmdMaxLen-3] + "..."
		}

		tw.AppendRow(table.Row{
			c.Name,
			c.Image,
			command,
			c.Service,
			created,
			status,
			portStr,
		})
	}

	logger.Print(tw.Render())
}

func formatPorts(ports []PortInfo, exposedPorts []string) string {
	var parts []string
	for _, ep := range exposedPorts {
		parts = append(parts, ep)
	}
	for _, p := range ports {
		ip := p.HostIP
		if ip == "" {
			ip = "0.0.0.0"
		}
		parts = append(parts, fmt.Sprintf("%s:%d->%d/%s", ip, p.HostPort, p.ContainerPort, p.Protocol))
	}
	return strings.Join(parts, ", ")
}

func formatCreated(createdAt, exitedAt time.Time, state string) string {
	if state == "exited" && !exitedAt.IsZero() {
		return formatTimeAgo(exitedAt)
	}
	return formatTimeAgo(createdAt)
}

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		return t.Format("Jan 02 2006")
	}
	if d < time.Minute {
		s := int(d.Seconds())
		if s == 0 {
			return "just now"
		}
		return fmt.Sprintf("%ds ago", s)
	}
	if d < time.Hour {
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	}
	if d < 7*24*time.Hour {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
	return t.Format("Jan 02 2006")
}
