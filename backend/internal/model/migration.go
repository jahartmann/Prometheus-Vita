package model

import (
	"time"

	"github.com/google/uuid"
)

type MigrationStatus string

const (
	MigrationStatusPending      MigrationStatus = "pending"
	MigrationStatusPreparing    MigrationStatus = "preparing"
	MigrationStatusBackingUp    MigrationStatus = "backing_up"
	MigrationStatusTransferring MigrationStatus = "transferring"
	MigrationStatusRestoring    MigrationStatus = "restoring"
	MigrationStatusCleaningUp   MigrationStatus = "cleaning_up"
	MigrationStatusCompleted    MigrationStatus = "completed"
	MigrationStatusFailed       MigrationStatus = "failed"
	MigrationStatusCancelled    MigrationStatus = "cancelled"
)

func (s MigrationStatus) IsTerminal() bool {
	return s == MigrationStatusCompleted || s == MigrationStatusFailed || s == MigrationStatusCancelled
}

type MigrationMode string

const (
	MigrationModeStop     MigrationMode = "stop"
	MigrationModeSnapshot MigrationMode = "snapshot"
	MigrationModeSuspend  MigrationMode = "suspend"
)

func (m MigrationMode) IsValid() bool {
	switch m {
	case MigrationModeStop, MigrationModeSnapshot, MigrationModeSuspend:
		return true
	}
	return false
}

type VMMigration struct {
	ID               uuid.UUID       `json:"id"`
	SourceNodeID     uuid.UUID       `json:"source_node_id"`
	TargetNodeID     uuid.UUID       `json:"target_node_id"`
	VMID             int             `json:"vmid"`
	VMName           string          `json:"vm_name"`
	VMType           string          `json:"vm_type"`
	Status           MigrationStatus `json:"status"`
	Mode             MigrationMode   `json:"mode"`
	TargetStorage    string          `json:"target_storage"`
	Progress         int             `json:"progress"`
	CurrentStep      string          `json:"current_step"`
	VzdumpFilePath   *string         `json:"vzdump_file_path,omitempty"`
	VzdumpFileSize   *int64          `json:"vzdump_file_size,omitempty"`
	VzdumpTaskUPID   *string         `json:"vzdump_task_upid,omitempty"`
	TransferBytesSent int64          `json:"transfer_bytes_sent"`
	TransferSpeedBps  int64          `json:"transfer_speed_bps"`
	NewVMID          *int            `json:"new_vmid,omitempty"`
	RestoreTaskUPID  *string         `json:"restore_task_upid,omitempty"`
	CleanupSource    bool            `json:"cleanup_source"`
	CleanupTarget    bool            `json:"cleanup_target"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	InitiatedBy      *uuid.UUID      `json:"initiated_by,omitempty"`
}

type StartMigrationRequest struct {
	SourceNodeID  uuid.UUID     `json:"source_node_id" validate:"required"`
	TargetNodeID  uuid.UUID     `json:"target_node_id" validate:"required"`
	VMID          int           `json:"vmid" validate:"required"`
	TargetStorage string        `json:"target_storage" validate:"required"`
	Mode          MigrationMode `json:"mode"`
	NewVMID       *int          `json:"new_vmid,omitempty"`
	CleanupSource bool          `json:"cleanup_source"`
	CleanupTarget bool          `json:"cleanup_target"`
}

type MigrationResponse struct {
	ID               uuid.UUID       `json:"id"`
	SourceNodeID     uuid.UUID       `json:"source_node_id"`
	TargetNodeID     uuid.UUID       `json:"target_node_id"`
	VMID             int             `json:"vmid"`
	VMName           string          `json:"vm_name"`
	VMType           string          `json:"vm_type"`
	Status           MigrationStatus `json:"status"`
	Mode             MigrationMode   `json:"mode"`
	TargetStorage    string          `json:"target_storage"`
	Progress         int             `json:"progress"`
	CurrentStep      string          `json:"current_step"`
	TransferBytesSent int64          `json:"transfer_bytes_sent"`
	TransferSpeedBps  int64          `json:"transfer_speed_bps"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

func (m *VMMigration) ToResponse() MigrationResponse {
	return MigrationResponse{
		ID:                m.ID,
		SourceNodeID:      m.SourceNodeID,
		TargetNodeID:      m.TargetNodeID,
		VMID:              m.VMID,
		VMName:            m.VMName,
		VMType:            m.VMType,
		Status:            m.Status,
		Mode:              m.Mode,
		TargetStorage:     m.TargetStorage,
		Progress:          m.Progress,
		CurrentStep:       m.CurrentStep,
		TransferBytesSent: m.TransferBytesSent,
		TransferSpeedBps:  m.TransferSpeedBps,
		ErrorMessage:      m.ErrorMessage,
		StartedAt:         m.StartedAt,
		CompletedAt:       m.CompletedAt,
		CreatedAt:         m.CreatedAt,
	}
}
