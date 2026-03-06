package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SecurityEvent struct {
	ID             uuid.UUID       `json:"id"`
	NodeID         uuid.UUID       `json:"node_id"`
	Category       string          `json:"category"`
	Severity       string          `json:"severity"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Impact         string          `json:"impact"`
	Recommendation string          `json:"recommendation"`
	Metrics        json.RawMessage `json:"metrics,omitempty"`
	AffectedVMs    []string        `json:"affected_vms,omitempty"`
	NodeName       string          `json:"node_name,omitempty"`
	IsAcknowledged bool            `json:"is_acknowledged"`
	DetectedAt     time.Time       `json:"detected_at"`
	AcknowledgedAt *time.Time      `json:"acknowledged_at,omitempty"`
	AnalysisModel  string          `json:"analysis_model,omitempty"`
}

type SecurityEventStats struct {
	Total          int            `json:"total"`
	Unacknowledged int            `json:"unacknowledged"`
	BySeverity     map[string]int `json:"by_severity"`
	ByCategory     map[string]int `json:"by_category"`
}
