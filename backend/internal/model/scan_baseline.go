package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ScanBaseline struct {
	ID            uuid.UUID       `json:"id"`
	NodeID        uuid.UUID       `json:"node_id"`
	Label         string          `json:"label,omitempty"`
	IsActive      bool            `json:"is_active"`
	BaselineJSON  json.RawMessage `json:"baseline_json"`
	WhitelistJSON json.RawMessage `json:"whitelist_json,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type CreateBaselineRequest struct {
	Label         string          `json:"label,omitempty"`
	WhitelistJSON json.RawMessage `json:"whitelist_json,omitempty"`
}

type UpdateBaselineRequest struct {
	Label         *string          `json:"label,omitempty"`
	WhitelistJSON *json.RawMessage `json:"whitelist_json,omitempty"`
}
