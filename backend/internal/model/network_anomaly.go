package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type NetworkAnomaly struct {
	ID             uuid.UUID       `json:"id"`
	NodeID         uuid.UUID       `json:"node_id"`
	AnomalyType    string          `json:"anomaly_type"`
	RiskScore      float64         `json:"risk_score"`
	DetailsJSON    json.RawMessage `json:"details_json"`
	ScanID         *uuid.UUID      `json:"scan_id,omitempty"`
	IsAcknowledged bool            `json:"is_acknowledged"`
	AcknowledgedAt *time.Time      `json:"acknowledged_at,omitempty"`
	AcknowledgedBy *uuid.UUID      `json:"acknowledged_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}
