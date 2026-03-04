package backup

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// cronField holds the set of allowed integer values for one field of a cron
// expression.
type cronField struct {
	values map[int]bool
}

// ParseCron validates a 5-field cron expression (minute hour day month weekday).
// Supported syntax per field: "*", a single number, "*/N" (step), "N-M"
// (range), and "N,M,..." (list). Returns an error if the expression is
// malformed or any value is out of range.
func ParseCron(expr string) error {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("cron expression must have exactly 5 fields, got %d", len(fields))
	}

	limits := []struct {
		name string
		min  int
		max  int
	}{
		{"minute", 0, 59},
		{"hour", 0, 23},
		{"day", 1, 31},
		{"month", 1, 12},
		{"weekday", 0, 6},
	}

	for i, field := range fields {
		if _, err := parseField(field, limits[i].min, limits[i].max); err != nil {
			return fmt.Errorf("invalid %s field %q: %w", limits[i].name, field, err)
		}
	}

	return nil
}

// NextRun calculates the next time a cron expression matches, starting from
// the given time (exclusive). It checks each minute up to 366 days ahead and
// returns an error if no match is found within that window.
func NextRun(expr string, after time.Time) (time.Time, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("cron expression must have exactly 5 fields, got %d", len(fields))
	}

	minuteField, err := parseField(fields[0], 0, 59)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute field: %w", err)
	}
	hourField, err := parseField(fields[1], 0, 23)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour field: %w", err)
	}
	dayField, err := parseField(fields[2], 1, 31)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day field: %w", err)
	}
	monthField, err := parseField(fields[3], 1, 12)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month field: %w", err)
	}
	weekdayField, err := parseField(fields[4], 0, 6)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid weekday field: %w", err)
	}

	// Start from the next minute after 'after'
	t := after.Truncate(time.Minute).Add(time.Minute)
	limit := after.Add(366 * 24 * time.Hour)

	for t.Before(limit) {
		if monthField.values[int(t.Month())] &&
			dayField.values[t.Day()] &&
			weekdayField.values[int(t.Weekday())] &&
			hourField.values[t.Hour()] &&
			minuteField.values[t.Minute()] {
			return t, nil
		}
		t = t.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("no matching time found within 366 days")
}

// parseField parses a single cron field string into a set of integer values
// within the given min..max range.
func parseField(field string, min, max int) (*cronField, error) {
	cf := &cronField{values: make(map[int]bool)}

	// Handle comma-separated lists
	parts := strings.Split(field, ",")
	for _, part := range parts {
		if err := parsePart(part, min, max, cf); err != nil {
			return nil, err
		}
	}

	return cf, nil
}

// parsePart handles a single component of a cron field, which may be "*",
// "*/N", "N-M", "N-M/S", or a plain number.
func parsePart(part string, min, max int, cf *cronField) error {
	// Handle step values (*/N or N-M/S)
	var step int
	if idx := strings.Index(part, "/"); idx != -1 {
		stepStr := part[idx+1:]
		var err error
		step, err = strconv.Atoi(stepStr)
		if err != nil || step <= 0 {
			return fmt.Errorf("invalid step value %q", stepStr)
		}
		part = part[:idx]
	}

	var start, end int

	switch {
	case part == "*":
		start = min
		end = max
	case strings.Contains(part, "-"):
		rangeParts := strings.SplitN(part, "-", 2)
		var err error
		start, err = strconv.Atoi(rangeParts[0])
		if err != nil {
			return fmt.Errorf("invalid range start %q", rangeParts[0])
		}
		end, err = strconv.Atoi(rangeParts[1])
		if err != nil {
			return fmt.Errorf("invalid range end %q", rangeParts[1])
		}
	default:
		val, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("invalid value %q", part)
		}
		if val < min || val > max {
			return fmt.Errorf("value %d out of range [%d-%d]", val, min, max)
		}
		if step > 0 {
			// e.g. "5/10" means starting at 5, every 10
			for v := val; v <= max; v += step {
				cf.values[v] = true
			}
		} else {
			cf.values[val] = true
		}
		return nil
	}

	// Validate range
	if start < min || start > max {
		return fmt.Errorf("start value %d out of range [%d-%d]", start, min, max)
	}
	if end < min || end > max {
		return fmt.Errorf("end value %d out of range [%d-%d]", end, min, max)
	}
	if start > end {
		return fmt.Errorf("range start %d > end %d", start, end)
	}

	if step == 0 {
		step = 1
	}
	for v := start; v <= end; v += step {
		cf.values[v] = true
	}

	return nil
}
