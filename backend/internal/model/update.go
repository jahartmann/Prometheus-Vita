package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type UpdateCheckStatus string

const (
	UpdateCheckPending   UpdateCheckStatus = "pending"
	UpdateCheckRunning   UpdateCheckStatus = "running"
	UpdateCheckCompleted UpdateCheckStatus = "completed"
	UpdateCheckFailed    UpdateCheckStatus = "failed"
)

type UpdateCheck struct {
	ID              uuid.UUID         `json:"id"`
	NodeID          uuid.UUID         `json:"node_id"`
	Status          UpdateCheckStatus `json:"status"`
	TotalUpdates    int               `json:"total_updates"`
	SecurityUpdates int               `json:"security_updates"`
	Packages        json.RawMessage   `json:"packages,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	CheckedAt       time.Time         `json:"checked_at"`
	CreatedAt       time.Time         `json:"created_at"`
}

type PackageUpdate struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	NewVersion     string `json:"new_version"`
	IsSecurity     bool   `json:"is_security"`
	Source         string `json:"source,omitempty"`
}
