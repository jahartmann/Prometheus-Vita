package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type NetworkScan struct {
	ID          uuid.UUID       `json:"id"`
	NodeID      uuid.UUID       `json:"node_id"`
	ScanType    string          `json:"scan_type"`
	ResultsJSON json.RawMessage `json:"results_json"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

type TriggerScanRequest struct {
	ScanType string `json:"scan_type" validate:"required,oneof=quick full"`
}
