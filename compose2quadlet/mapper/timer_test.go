package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestTimers_Basic(t *testing.T) {
	jobs := types.Jobs{
		"cleanup": types.JobConfig{
			Name: "cleanup",
			Triggers: &types.TriggerConfig{
				Schedule: []types.ScheduleConfig{
					{Cron: "0 3 * * *"},
				},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Timers(jobs, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 timer unit, got %d", len(units))
	}

	u := units[0]
	if u.Type != c2qtypes.UnitTimer {
		t.Fatalf("expected timer unit type, got %s", u.Type)
	}
	if u.Name != "cleanup" {
		t.Fatalf("expected timer name 'cleanup', got %s", u.Name)
	}

	// Check [Unit] section
	unitSec := findSection(u, c2qtypes.SectionUnit)
	if unitSec == nil {
		t.Fatal("expected [Unit] section")
	}
	assertDirective(t, unitSec.Directives, "Description", "Timer for job cleanup")

	// Check [Timer] section
	timerSec := findSection(u, c2qtypes.SectionTimer)
	if timerSec == nil {
		t.Fatal("expected [Timer] section")
	}
	assertDirective(t, timerSec.Directives, "OnCalendar", "*-*-* 03:00:00")

	// Check [Install] section
	installSec := findSection(u, c2qtypes.SectionInstall)
	if installSec == nil {
		t.Fatal("expected [Install] section")
	}
	assertDirective(t, installSec.Directives, "WantedBy", "timers.target")
}

func TestTimers_WithTimezone(t *testing.T) {
	jobs := types.Jobs{
		"backup": types.JobConfig{
			Name: "backup",
			Triggers: &types.TriggerConfig{
				Schedule: []types.ScheduleConfig{
					{
						Cron:     "0 0 * * *",
						Timezone: "Europe/Paris",
					},
				},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Timers(jobs, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 timer unit, got %d", len(units))
	}

	timerSec := findSection(units[0], c2qtypes.SectionTimer)
	if timerSec == nil {
		t.Fatal("expected [Timer] section")
	}
	assertDirective(t, timerSec.Directives, "OnCalendar", "Europe/Paris *-*-* 00:00:00")
}

func TestTimers_WithMissedFires(t *testing.T) {
	tests := []struct {
		name           string
		missedFires    string
		expectPersistent bool
	}{
		{
			name:           "missed_fires=one",
			missedFires:    "one",
			expectPersistent: true,
		},
		{
			name:           "missed_fires=skip",
			missedFires:    "skip",
			expectPersistent: false,
		},
		{
			name:           "missed_fires empty defaults to one",
			missedFires:    "",
			expectPersistent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobs := types.Jobs{
				"job": types.JobConfig{
					Name: "job",
					Triggers: &types.TriggerConfig{
						Schedule: []types.ScheduleConfig{
							{
								Cron:        "0 0 * * *",
								MissedFires: tt.missedFires,
							},
						},
					},
				},
			}
			cfg := c2qtypes.DefaultConfig()
			units := Timers(jobs, cfg)

			if len(units) != 1 {
				t.Fatalf("expected 1 timer unit, got %d", len(units))
			}

			timerSec := findSection(units[0], c2qtypes.SectionTimer)
			if timerSec == nil {
				t.Fatal("expected [Timer] section")
			}

			hasPersistent := hasDirective(timerSec.Directives, "Persistent", "true")
			if tt.expectPersistent && !hasPersistent {
				t.Fatal("expected Persistent=true directive")
			}
			if !tt.expectPersistent && hasPersistent {
				t.Fatal("did not expect Persistent directive")
			}
		})
	}
}

func TestTimers_MultipleSchedules(t *testing.T) {
	jobs := types.Jobs{
		"multi": types.JobConfig{
			Name: "multi",
			Triggers: &types.TriggerConfig{
				Schedule: []types.ScheduleConfig{
					{Cron: "0 3 * * *"},
					{Cron: "0 15 * * *"},
				},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Timers(jobs, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 timer unit, got %d", len(units))
	}

	timerSec := findSection(units[0], c2qtypes.SectionTimer)
	if timerSec == nil {
		t.Fatal("expected [Timer] section")
	}

	// Should have two OnCalendar directives
	count := 0
	for _, d := range timerSec.Directives {
		if d.Key == "OnCalendar" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 OnCalendar directives, got %d", count)
	}

	assertDirective(t, timerSec.Directives, "OnCalendar", "*-*-* 03:00:00")
	assertDirective(t, timerSec.Directives, "OnCalendar", "*-*-* 15:00:00")
}

func TestTimers_NoTriggers(t *testing.T) {
	manual := true
	jobs := types.Jobs{
		"manual": types.JobConfig{
			Name: "manual",
			Triggers: &types.TriggerConfig{
				Manual: &manual,
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Timers(jobs, cfg)

	if len(units) != 0 {
		t.Fatalf("expected 0 timer units for manual job, got %d", len(units))
	}
}

func TestTimers_EmptySchedule(t *testing.T) {
	jobs := types.Jobs{
		"job": types.JobConfig{
			Name: "job",
			Triggers: &types.TriggerConfig{
				Schedule: []types.ScheduleConfig{},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Timers(jobs, cfg)

	if len(units) != 0 {
		t.Fatalf("expected 0 timer units for empty schedule, got %d", len(units))
	}
}

func TestTimers_InvalidCron(t *testing.T) {
	jobs := types.Jobs{
		"bad": types.JobConfig{
			Name: "bad",
			Triggers: &types.TriggerConfig{
				Schedule: []types.ScheduleConfig{
					{Cron: "invalid cron"},
				},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Timers(jobs, cfg)

	// Should skip invalid cron and produce no timer
	if len(units) != 0 {
		t.Fatalf("expected 0 timer units for invalid cron, got %d", len(units))
	}

	// Should have a warning
	if !hasWarning(cfg, "bad", "triggers.schedule.cron") {
		t.Fatal("expected warning for invalid cron")
	}
}

func TestTimers_MultipleJobs(t *testing.T) {
	jobs := types.Jobs{
		"job1": types.JobConfig{
			Name: "job1",
			Triggers: &types.TriggerConfig{
				Schedule: []types.ScheduleConfig{
					{Cron: "0 3 * * *"},
				},
			},
		},
		"job2": types.JobConfig{
			Name: "job2",
			Triggers: &types.TriggerConfig{
				Schedule: []types.ScheduleConfig{
					{Cron: "0 15 * * *"},
				},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := Timers(jobs, cfg)

	if len(units) != 2 {
		t.Fatalf("expected 2 timer units, got %d", len(units))
	}

	// Check that both jobs have timers
	names := make(map[string]bool)
	for _, u := range units {
		names[u.Name] = true
	}
	if !names["job1"] || !names["job2"] {
		t.Fatal("expected timers for both job1 and job2")
	}
}

func findSection(u c2qtypes.QuadletUnit, sectionName string) *c2qtypes.Section {
	for i := range u.Sections {
		if u.Sections[i].Name == sectionName {
			return &u.Sections[i]
		}
	}
	return nil
}
