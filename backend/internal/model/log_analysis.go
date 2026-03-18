package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type LogPattern struct {
	Pattern     string `json:"pattern"`
	Occurrences int    `json:"occurrences"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

type LogAnalysis struct {
	ID         uuid.UUID       `json:"id"`
	NodeIDs    []uuid.UUID     `json:"node_ids"`
	TimeFrom   time.Time       `json:"time_from"`
	TimeTo     time.Time       `json:"time_to"`
	ReportJSON json.RawMessage `json:"report_json"`
	ModelUsed  string          `json:"model_used"`
	ScheduleID *uuid.UUID      `json:"schedule_id,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type LogAnalysisReport struct {
	Summary             string       `json:"summary"`
	Anomalies           []LogAnomaly `json:"anomalies"`
	Patterns            []LogPattern `json:"patterns"`
	RootCauseHypotheses []string     `json:"root_cause_hypotheses"`
	Recommendations     []string     `json:"recommendations"`
	TimeRange           TimeRange    `json:"time_range"`
	NodesAnalyzed       []string     `json:"nodes_analyzed"`
	ModelUsed           string       `json:"model_used"`
}

type AnalyzeLogsRequest struct {
	NodeIDs  []uuid.UUID `json:"node_ids" validate:"required"`
	TimeFrom time.Time   `json:"time_from" validate:"required"`
	TimeTo   time.Time   `json:"time_to" validate:"required"`
	Context  string      `json:"context,omitempty"`
}
