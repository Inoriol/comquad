package orchestrator

import (
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
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
		fmt.Printf("No containers found for project %s\n", o.projectName)
		return nil
	}

	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	var unitNames []string
	for _, c := range containers {
		unitNames = append(unitNames, c.Name+".service")
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
			unitName := containers[i].Name + ".service"
			if u, ok := unitStateMap[unitName]; ok {
				containers[i].DBusActive = u.ActiveState
				containers[i].DBusSub = u.SubState
			}
		}
	}

	// Print table
	printPsTable(containers)

	return nil
}

func printPsTable(containers []ContainerInfo) {
	nameW := max(len("NAME"), 20)
	imageW := max(len("IMAGE"), 30)
	commandW := max(len("COMMAND"), 25)
	serviceW := max(len("SERVICE"), 12)
	createdW := max(len("CREATED"), 20)
	statusW := max(len("STATUS"), 20)
	portsW := max(len("PORTS"), 30)

	for _, c := range containers {
		if len(c.Name) > nameW {
			nameW = len(c.Name)
		}
		if len(c.Image) > imageW {
			imageW = len(c.Image)
		}
		if len(c.Command) > commandW {
			commandW = len(c.Command)
		}
		if len(c.Service) > serviceW {
			serviceW = len(c.Service)
		}
		if len(c.Status) > statusW {
			statusW = len(c.Status)
		}
		portStr := formatPorts(c.Ports, c.ExposedPorts)
		if len(portStr) > portsW {
			portsW = len(portStr)
		}
	}

	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		nameW, "NAME",
		imageW, "IMAGE",
		commandW, "COMMAND",
		serviceW, "SERVICE",
		createdW, "CREATED",
		statusW, "STATUS",
		portsW, "PORTS")
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)))

	for _, c := range containers {
		status := c.Status
		if c.State == "exited" && c.ExitCode != 0 {
			status = fmt.Sprintf("Exited (%d) %s", c.ExitCode, c.Status)
		} else if c.State == "dead" {
			status = "Dead"
		}

		created := formatCreated(c.CreatedAt, c.ExitedAt, c.State)
		portStr := formatPorts(c.Ports, c.ExposedPorts)

		row := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			nameW, truncate(c.Name, nameW),
			imageW, truncate(c.Image, imageW),
			commandW, truncate(c.Command, commandW),
			serviceW, truncate(c.Service, serviceW),
			createdW, created,
			statusW, truncate(status, statusW),
			portsW, truncate(portStr, portsW))
		fmt.Println(row)
	}
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
