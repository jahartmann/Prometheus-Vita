package model

import (
	"time"
)

type LogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	NodeID    string    `json:"node_id"`
	Source    string    `json:"source"`
	Severity  string    `json:"severity"`
	Process   string    `json:"process"`
	PID       int       `json:"pid"`
	Message   string    `json:"message"`
	Raw       string    `json:"raw"`
}

type LogAssessment struct {
	Severity     string  `json:"severity"`
	AnomalyScore float64 `json:"anomaly_score"`
	Category     string  `json:"category"`
	Summary      string  `json:"summary"`
}

type AnnotatedLogEntry struct {
	LogEntry
	Assessment *LogAssessment `json:"assessment,omitempty"`
}
