package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLogEntry struct {
	ID          uuid.UUID       `json:"id"`
	UserID      *uuid.UUID      `json:"user_id,omitempty"`
	Username    string          `json:"username,omitempty"`
	APITokenID  *uuid.UUID      `json:"api_token_id,omitempty"`
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	StatusCode  int             `json:"status_code"`
	IPAddress   string          `json:"ip_address,omitempty"`
	UserAgent   string          `json:"user_agent,omitempty"`
	RequestBody json.RawMessage `json:"request_body,omitempty"`
	DurationMS  int             `json:"duration_ms"`
	CreatedAt   time.Time       `json:"created_at"`
}
