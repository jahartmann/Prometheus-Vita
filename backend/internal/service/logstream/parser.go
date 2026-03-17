package logstream

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/model"
)

var syslogRegex = regexp.MustCompile(`^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+(\S+?)(?:\[(\d+)\])?:\s+(.*)$`)

// ParseSyslog parses a standard syslog line into a structured LogEntry.
func ParseSyslog(raw, nodeID, source string) model.LogEntry {
	sanitized := Sanitize(raw)
	entry := model.LogEntry{
		ID:     fmt.Sprintf("%s-%d", nodeID, time.Now().UnixNano()),
		NodeID: nodeID,
		Source: source,
		Raw:    sanitized,
	}

	matches := syslogRegex.FindStringSubmatch(sanitized)
	if matches != nil {
		entry.Timestamp = ParseTimestamp(matches[1])
		// matches[2] is hostname — not stored separately in the model
		entry.Process = matches[3]
		if matches[4] != "" {
			pid, err := strconv.Atoi(matches[4])
			if err == nil {
				entry.PID = pid
			}
		}
		entry.Message = matches[5]
	} else {
		entry.Timestamp = time.Now()
		entry.Message = sanitized
	}

	entry.Severity = InferSeverity(entry.Message)
	return entry
}

// ParseTimestamp parses syslog and ISO 8601 timestamp strings.
// Falls back to time.Now() if parsing fails.
func ParseTimestamp(raw string) time.Time {
	// Try ISO 8601 first
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}

	// Try syslog format: "Jan  2 15:04:05" (assumes current year)
	syslogFormat := "Jan  2 15:04:05"
	syslogFormatSingle := "Jan 2 15:04:05"
	year := time.Now().Year()
	withYear := fmt.Sprintf("%d %s", year, raw)
	if t, err := time.Parse("2006 "+syslogFormat, withYear); err == nil {
		return t
	}
	if t, err := time.Parse("2006 "+syslogFormatSingle, withYear); err == nil {
		return t
	}

	return time.Now()
}

// InferSeverity performs keyword-based severity inference on a log message.
func InferSeverity(message string) string {
	lower := strings.ToLower(message)

	criticalKeywords := []string{"emerg", "panic", "fatal"}
	for _, kw := range criticalKeywords {
		if strings.Contains(lower, kw) {
			return "critical"
		}
	}

	errorKeywords := []string{"error", "err", "fail", "failed", "failure"}
	for _, kw := range errorKeywords {
		if strings.Contains(lower, kw) {
			return "error"
		}
	}

	warningKeywords := []string{"warn", "warning"}
	for _, kw := range warningKeywords {
		if strings.Contains(lower, kw) {
			return "warning"
		}
	}

	infoKeywords := []string{"info", "notice"}
	for _, kw := range infoKeywords {
		if strings.Contains(lower, kw) {
			return "info"
		}
	}

	return "debug"
}
