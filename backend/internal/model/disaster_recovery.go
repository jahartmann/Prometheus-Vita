package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Node profile - hardware/software inventory

type NodeProfile struct {
	ID                uuid.UUID       `json:"id"`
	NodeID            uuid.UUID       `json:"node_id"`
	CollectedAt       time.Time       `json:"collected_at"`
	CPUModel          string          `json:"cpu_model,omitempty"`
	CPUCores          int             `json:"cpu_cores,omitempty"`
	CPUThreads        int             `json:"cpu_threads,omitempty"`
	MemoryTotalBytes  int64           `json:"memory_total_bytes,omitempty"`
	MemoryModules     json.RawMessage `json:"memory_modules,omitempty"`
	Disks             json.RawMessage `json:"disks,omitempty"`
	NetworkInterfaces json.RawMessage `json:"network_interfaces,omitempty"`
	PVEVersion        string          `json:"pve_version,omitempty"`
	KernelVersion     string          `json:"kernel_version,omitempty"`
	InstalledPackages json.RawMessage `json:"installed_packages,omitempty"`
	StorageLayout     json.RawMessage `json:"storage_layout,omitempty"`
	CustomData        json.RawMessage `json:"custom_data,omitempty"`
}

// DR readiness score

type DRReadinessScore struct {
	ID           uuid.UUID       `json:"id"`
	NodeID       uuid.UUID       `json:"node_id"`
	OverallScore int             `json:"overall_score"`
	BackupScore  int             `json:"backup_score"`
	ProfileScore int             `json:"profile_score"`
	ConfigScore  int             `json:"config_score"`
	Details      json.RawMessage `json:"details,omitempty"`
	CalculatedAt time.Time       `json:"calculated_at"`
}

// Recovery runbooks

type RunbookStep struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Command        string `json:"command,omitempty"`
	ExpectedOutput string `json:"expected_output,omitempty"`
	IsManual       bool   `json:"is_manual"`
}

type RecoveryRunbook struct {
	ID          uuid.UUID       `json:"id"`
	NodeID      *uuid.UUID      `json:"node_id,omitempty"`
	Title       string          `json:"title"`
	Scenario    string          `json:"scenario"`
	Steps       json.RawMessage `json:"steps"`
	IsTemplate  bool            `json:"is_template"`
	GeneratedAt time.Time       `json:"generated_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// DR simulation

type DRSimulationRequest struct {
	NodeID   uuid.UUID `json:"node_id" validate:"required"`
	Scenario string    `json:"scenario" validate:"required"`
}

type DRSimulationResult struct {
	NodeID       uuid.UUID              `json:"node_id"`
	Scenario     string                 `json:"scenario"`
	Ready        bool                   `json:"ready"`
	Checks       []DRSimulationCheck    `json:"checks"`
	Summary      string                 `json:"summary"`
}

type DRSimulationCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// DTOs

type CollectProfileRequest struct {
	NodeID uuid.UUID `json:"node_id"`
}

type GenerateRunbookRequest struct {
	Scenario string `json:"scenario" validate:"required"`
}

type UpdateRunbookRequest struct {
	Title *string          `json:"title,omitempty"`
	Steps *json.RawMessage `json:"steps,omitempty"`
}
