package mapper

import (
	"fmt"
	"strings"
)

func CronToSystemdCalendar(cron string) (string, error) {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return "", fmt.Errorf("invalid cron expression: expected 5 fields, got %d", len(fields))
	}

	minute, hour, dom, month, dow := fields[0], fields[1], fields[2], fields[3], fields[4]

	var parts []string

	if dom != "*" || month != "*" || dow != "*" {
		datePart := convertDatePart(dom, month, dow)
		if datePart != "" {
			parts = append(parts, datePart)
		}
	}

	timePart := convertTimePart(hour, minute)
	if timePart != "" {
		parts = append(parts, timePart)
	}

	if len(parts) == 0 {
		return "*-*-* *:*:00", nil
	}

	if len(parts) == 1 {
		if strings.Contains(parts[0], ":") {
			return "*-*-* " + parts[0], nil
		}
		return parts[0] + " 00:00:00", nil
	}

	return parts[0] + " " + parts[1], nil
}

func convertTimePart(hour, minute string) string {
	h := convertField(hour, 0, 23)
	m := convertField(minute, 0, 59)
	return fmt.Sprintf("%s:%s:00", h, m)
}

func convertDatePart(dom, month, dow string) string {
	var parts []string

	if month != "*" {
		parts = append(parts, convertMonth(month))
	}

	if dom != "*" && dow != "*" {
		d := convertField(dom, 1, 31)
		w := convertDow(dow)
		parts = append(parts, fmt.Sprintf("%s %s", d, w))
	} else if dom != "*" {
		parts = append(parts, convertField(dom, 1, 31))
	} else if dow != "*" {
		parts = append(parts, convertDow(dow))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}

func convertField(field string, min, max int) string {
	if field == "*" {
		return "*"
	}

	if strings.Contains(field, "/") {
		parts := strings.SplitN(field, "/", 2)
		base := parts[0]
		step := parts[1]
		if base == "*" {
			return fmt.Sprintf("*..*%s/%s", "", step)
		}
		return fmt.Sprintf("%s/%s", convertRange(base, min, max), step)
	}

	if strings.Contains(field, ",") {
		var converted []string
		for _, part := range strings.Split(field, ",") {
			converted = append(converted, convertRange(part, min, max))
		}
		return strings.Join(converted, ",")
	}

	return convertRange(field, min, max)
}

func convertRange(field string, min, max int) string {
	if strings.Contains(field, "-") {
		parts := strings.SplitN(field, "-", 2)
		return parts[0] + ".." + parts[1]
	}
	return field
}

func convertMonth(month string) string {
	monthNames := map[string]string{
		"1": "01", "2": "02", "3": "03", "4": "04",
		"5": "05", "6": "06", "7": "07", "8": "08",
		"9": "09", "10": "10", "11": "11", "12": "12",
		"jan": "01", "feb": "02", "mar": "03", "apr": "04",
		"may": "05", "jun": "06", "jul": "07", "aug": "08",
		"sep": "09", "oct": "10", "nov": "11", "dec": "12",
	}

	if strings.Contains(month, ",") {
		var converted []string
		for _, part := range strings.Split(month, ",") {
			part = strings.ToLower(strings.TrimSpace(part))
			if name, ok := monthNames[part]; ok {
				converted = append(converted, name)
			} else {
				converted = append(converted, convertRange(part, 1, 12))
			}
		}
		return strings.Join(converted, ",")
	}

	month = strings.ToLower(month)
	if name, ok := monthNames[month]; ok {
		return name
	}
	return convertRange(month, 1, 12)
}

func convertDow(dow string) string {
	dowNames := map[string]string{
		"0": "Sun", "1": "Mon", "2": "Tue", "3": "Wed",
		"4": "Thu", "5": "Fri", "6": "Sat", "7": "Sun",
		"sun": "Sun", "mon": "Mon", "tue": "Tue", "wed": "Wed",
		"thu": "Thu", "fri": "Fri", "sat": "Sat",
	}

	if strings.Contains(dow, ",") {
		var converted []string
		for _, part := range strings.Split(dow, ",") {
			part = strings.ToLower(strings.TrimSpace(part))
			if name, ok := dowNames[part]; ok {
				converted = append(converted, name)
			} else {
				converted = append(converted, part)
			}
		}
		return strings.Join(converted, ",")
	}

	if strings.Contains(dow, "-") {
		parts := strings.SplitN(dow, "-", 2)
		start := convertSingleDow(parts[0])
		end := convertSingleDow(parts[1])
		return start + ".." + end
	}

	dow = strings.ToLower(dow)
	if name, ok := dowNames[dow]; ok {
		return name
	}
	return dow
}

func convertSingleDow(d string) string {
	dowNames := map[string]string{
		"0": "Sun", "1": "Mon", "2": "Tue", "3": "Wed",
		"4": "Thu", "5": "Fri", "6": "Sat", "7": "Sun",
		"sun": "Sun", "mon": "Mon", "tue": "Tue", "wed": "Wed",
		"thu": "Thu", "fri": "Fri", "sat": "Sat",
	}
	d = strings.ToLower(strings.TrimSpace(d))
	if name, ok := dowNames[d]; ok {
		return name
	}
	return d
}
