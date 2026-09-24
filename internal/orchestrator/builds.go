package orchestrator

import (
	"fmt"
	"strings"

	"github.com/Inoriol/comquad/internal/logger"
	"github.com/Inoriol/comquad/internal/output"
)

func (o *Orchestrator) handleBuilds(projectFiles []string) error {
	dbusMgr, err := o.newSystemd()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer dbusMgr.Close()

	for _, f := range projectFiles {
		if !strings.HasSuffix(f, ".build") {
			continue
		}

		unitName := BuildFileToUnitName(f)

		if err := dbusMgr.WaitForUnit(unitName, startUnitWaitTime); err != nil {
			logger.Warn(fmt.Sprintf("build unit %s not produced by quadlet generator, skipping: %v", unitName, err))
			continue
		}

		if !output.IsJSONMode() {
			logger.Action("Building image: " + unitName)
		}

		if err := dbusMgr.StopUnit(unitName); err != nil {
			if !output.IsJSONMode() {
				logger.Info(fmt.Sprintf("build unit %s was not running: %v", unitName, err))
			}
		}

		if err := dbusMgr.StartUnit(unitName); err != nil {
			return fmt.Errorf("failed to build image unit %s: %w", unitName, err)
		}

		if !output.IsJSONMode() {
			logger.Success("Built image: " + unitName)
		}
	}

	return nil
}
