package mapper

import (
	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func Timers(jobs types.Jobs, cfg *c2qtypes.Config) []c2qtypes.QuadletUnit {
	var units []c2qtypes.QuadletUnit

	for name, job := range jobs {
		if job.Triggers == nil || len(job.Triggers.Schedule) == 0 {
			continue
		}

		var sections []c2qtypes.Section

		var unitDirs []c2qtypes.Directive
		unitDirs = append(unitDirs, c2qtypes.Directive{Key: "Description", Values: []string{"Timer for job " + name}})

		if len(unitDirs) > 0 {
			sections = append(sections, c2qtypes.Section{Name: c2qtypes.SectionUnit, Directives: unitDirs})
		}

		var timerDirs []c2qtypes.Directive

		for _, schedule := range job.Triggers.Schedule {
			calendar, err := CronToSystemdCalendar(schedule.Cron)
			if err != nil {
				cfg.Warn(c2qtypes.Warning{
					Level:   c2qtypes.WarningSkipped,
					Service: name,
					Field:   "triggers.schedule.cron",
					Message: "invalid cron expression: " + err.Error(),
				})
				continue
			}

			if schedule.Timezone != "" {
				timerDirs = append(timerDirs, c2qtypes.Directive{Key: "OnCalendar", Values: []string{schedule.Timezone + " " + calendar}})
			} else {
				timerDirs = append(timerDirs, c2qtypes.Directive{Key: "OnCalendar", Values: []string{calendar}})
			}
		}

		if len(timerDirs) == 0 {
			continue
		}

		hasMissedFiresOne := false
		for _, schedule := range job.Triggers.Schedule {
			if schedule.MissedFires == "" || schedule.MissedFires == "one" {
				hasMissedFiresOne = true
				break
			}
		}
		if hasMissedFiresOne {
			timerDirs = append(timerDirs, c2qtypes.Directive{Key: "Persistent", Values: []string{"true"}})
		}

		sections = append(sections, c2qtypes.Section{Name: c2qtypes.SectionTimer, Directives: timerDirs})

		var installDirs []c2qtypes.Directive
		installDirs = append(installDirs, c2qtypes.Directive{Key: "WantedBy", Values: []string{"timers.target"}})
		sections = append(sections, c2qtypes.Section{Name: c2qtypes.SectionInstall, Directives: installDirs})

		units = append(units, c2qtypes.QuadletUnit{
			Type:     c2qtypes.UnitTimer,
			Name:     name,
			Sections: sections,
		})
	}

	return units
}
