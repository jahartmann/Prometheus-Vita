package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type LogAnomaly struct {
	ID             uuid.UUID  `json:"id"`
	NodeID         uuid.UUID  `json:"node_id"`
	Timestamp      time.Time  `json:"timestamp"`
	Source         string     `json:"source"`
	Severity       string     `json:"severity"`
	AnomalyScore   float64    `json:"anomaly_score"`
	Category       string     `json:"category"`
	Summary        string     `json:"summary"`
	RawLog         string     `json:"raw_log"`
	IsAcknowledged bool       `json:"is_acknowledged"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy *uuid.UUID `json:"acknowledged_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type LogBookmark struct {
	ID           uuid.UUID       `json:"id"`
	NodeID       uuid.UUID       `json:"node_id"`
	AnomalyID    *uuid.UUID      `json:"anomaly_id,omitempty"`
	LogEntryJSON json.RawMessage `json:"log_entry_json"`
	UserNote     string          `json:"user_note,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type CreateLogBookmarkRequest struct {
	NodeID       uuid.UUID       `json:"node_id" validate:"required"`
	AnomalyID    *uuid.UUID      `json:"anomaly_id,omitempty"`
	LogEntryJSON json.RawMessage `json:"log_entry_json" validate:"required"`
	UserNote     string          `json:"user_note,omitempty"`
}
