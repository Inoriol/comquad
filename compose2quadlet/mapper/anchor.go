package mapper

import (
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
)

func SelectAnchor(services map[string]types.ServiceConfig) string {
	if len(services) == 0 {
		return ""
	}

	var roots []string
	for name, svc := range services {
		if len(svc.DependsOn) == 0 {
			roots = append(roots, name)
		}
	}

	if len(roots) == 0 {
		return alphabeticallyFirst(services)
	}

	sort.Strings(roots)
	return roots[0]
}

func alphabeticallyFirst(services map[string]types.ServiceConfig) string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}
