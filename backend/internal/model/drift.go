package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DriftStatus string

const (
	DriftStatusPending   DriftStatus = "pending"
	DriftStatusRunning   DriftStatus = "running"
	DriftStatusCompleted DriftStatus = "completed"
	DriftStatusFailed    DriftStatus = "failed"
)

type DriftCheck struct {
	ID           uuid.UUID       `json:"id"`
	NodeID       uuid.UUID       `json:"node_id"`
	Status       DriftStatus     `json:"status"`
	TotalFiles   int             `json:"total_files"`
	ChangedFiles int             `json:"changed_files"`
	AddedFiles   int             `json:"added_files"`
	RemovedFiles int             `json:"removed_files"`
	Details      json.RawMessage `json:"details,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
	CheckedAt    time.Time       `json:"checked_at"`
	CreatedAt    time.Time       `json:"created_at"`
}

type DriftFileDetail struct {
	FilePath string `json:"file_path"`
	Status   string `json:"status"` // added, removed, modified, unchanged
	Diff     string `json:"diff,omitempty"`
}
