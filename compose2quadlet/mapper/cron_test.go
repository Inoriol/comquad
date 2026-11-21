package mapper

import (
	"testing"
)

func TestCronToSystemdCalendar_Basic(t *testing.T) {
	tests := []struct {
		name     string
		cron     string
		expected string
	}{
		{
			name:     "every minute",
			cron:     "* * * * *",
			expected: "*-*-* *:*:00",
		},
		{
			name:     "daily at midnight",
			cron:     "0 0 * * *",
			expected: "*-*-* 00:00:00",
		},
		{
			name:     "daily at 3am",
			cron:     "0 3 * * *",
			expected: "*-*-* 03:00:00",
		},
		{
			name:     "hourly",
			cron:     "0 * * * *",
			expected: "*-*-* *:00:00",
		},
		{
			name:     "every 15 minutes",
			cron:     "00/15 * * * *",
			expected: "*-*-* *:00/15:00",
		},
		{
			name:     "specific time",
			cron:     "30 14 * * *",
			expected: "*-*-* 14:30:00",
		},
		{
			name:     "weekdays at 9am",
			cron:     "0 9 * * 1-5",
			expected: "*-*-* Mon..Fri 09:00:00",
		},
		{
			name:     "sundays at noon",
			cron:     "0 12 * * 0",
			expected: "*-*-* Sun 12:00:00",
		},
		{
			name:     "first day of month",
			cron:     "0 0 1 * *",
			expected: "*-*-01 00:00:00",
		},
		{
			name:     "specific month",
			cron:     "0 0 1 6 *",
			expected: "*-06-01 00:00:00",
		},
		{
			name:     "multiple days",
			cron:     "0 0 1,15 * *",
			expected: "*-*-01,15 00:00:00",
		},
		{
			name:     "day range",
			cron:     "0 0 1-7 * *",
			expected: "*-*-01..07 00:00:00",
		},
		{
			name:     "multiple months",
			cron:     "0 0 1 1,6 *",
			expected: "*-01,06-01 00:00:00",
		},
		{
			name:     "month names",
			cron:     "0 0 1 jan,jun *",
			expected: "*-01,06-01 00:00:00",
		},
		{
			name:     "day of week names",
			cron:     "0 0 * * mon,wed,fri",
			expected: "*-*-* Mon,Wed,Fri 00:00:00",
		},
		{
			name:     "complex expression",
			cron:     "30 2 15 3 *",
			expected: "*-03-15 02:30:00",
		},
		{
			name:     "step in hour",
			cron:     "0 */2 * * *",
			expected: "*-*-* 00/2:00:00",
		},
		{
			name:     "step in minute",
			cron:     "*/5 * * * *",
			expected: "*-*-* *:00/5:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CronToSystemdCalendar(tt.cron)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCronToSystemdCalendar_Invalid(t *testing.T) {
	tests := []struct {
		name string
		cron string
	}{
		{
			name: "too few fields",
			cron: "* * *",
		},
		{
			name: "too many fields",
			cron: "* * * * * *",
		},
		{
			name: "empty string",
			cron: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CronToSystemdCalendar(tt.cron)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestConvertMonth(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1", "01"},
		{"12", "12"},
		{"jan", "01"},
		{"JAN", "01"},
		{"dec", "12"},
		{"DEC", "12"},
		{"1,6", "01,06"},
		{"jan,jun", "01,06"},
		{"1-6", "01..06"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertMonth(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertDow(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0", "Sun"},
		{"1", "Mon"},
		{"7", "Sun"},
		{"sun", "Sun"},
		{"SUN", "Sun"},
		{"mon", "Mon"},
		{"1-5", "Mon..Fri"},
		{"mon,fri", "Mon,Fri"},
		{"1,3,5", "Mon,Wed,Fri"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertDow(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		min      int
		max      int
		expected string
	}{
		{"wildcard", "*", 0, 59, "*"},
		{"single value", "30", 0, 59, "30"},
		{"range", "1-5", 1, 31, "01..05"},
		{"list", "1,15,30", 1, 31, "01,15,30"},
		{"step from wildcard", "00/15", 0, 59, "00/15"},
		{"step from value", "0/10", 0, 59, "0/10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertField(tt.field, tt.min, tt.max)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
