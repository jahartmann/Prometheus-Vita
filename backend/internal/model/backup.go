package model

import (
	"time"

	"github.com/google/uuid"
)

// Enums

type BackupType string

const (
	BackupTypeManual    BackupType = "manual"
	BackupTypeScheduled BackupType = "scheduled"
	BackupTypePreUpdate BackupType = "pre_update"
)

type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
)

// Domain models

type ConfigBackup struct {
	ID           uuid.UUID    `json:"id"`
	NodeID       uuid.UUID    `json:"node_id"`
	Version      int          `json:"version"`
	BackupType   BackupType   `json:"backup_type"`
	FileCount    int          `json:"file_count"`
	TotalSize    int64        `json:"total_size"`
	Status       BackupStatus `json:"status"`
	ErrorMessage  string       `json:"error_message,omitempty"`
	Notes         string       `json:"notes,omitempty"`
	RecoveryGuide string       `json:"recovery_guide,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	CompletedAt   *time.Time   `json:"completed_at,omitempty"`
}

type BackupFile struct {
	ID               uuid.UUID `json:"id"`
	BackupID         uuid.UUID `json:"backup_id"`
	FilePath         string    `json:"file_path"`
	FileHash         string    `json:"file_hash"`
	FileSize         int64     `json:"file_size"`
	FilePermissions  string    `json:"file_permissions,omitempty"`
	FileOwner        string    `json:"file_owner,omitempty"`
	Content          []byte    `json:"-"`
	DiffFromPrevious string    `json:"diff_from_previous,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type BackupSchedule struct {
	ID             uuid.UUID  `json:"id"`
	NodeID         uuid.UUID  `json:"node_id"`
	CronExpression string     `json:"cron_expression"`
	IsActive       bool       `json:"is_active"`
	RetentionCount int        `json:"retention_count"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// DTOs

type CreateBackupRequest struct {
	BackupType BackupType `json:"backup_type,omitempty"`
	Notes      string     `json:"notes,omitempty"`
}

type CreateScheduleRequest struct {
	CronExpression string `json:"cron_expression" validate:"required"`
	IsActive       bool   `json:"is_active"`
	RetentionCount int    `json:"retention_count,omitempty"`
}

type UpdateScheduleRequest struct {
	CronExpression *string `json:"cron_expression,omitempty"`
	IsActive       *bool   `json:"is_active,omitempty"`
	RetentionCount *int    `json:"retention_count,omitempty"`
}

type RestoreRequest struct {
	FilePaths []string `json:"file_paths"`
	DryRun    bool     `json:"dry_run"`
}

type RestorePreview struct {
	Files []RestoreFilePreview `json:"files"`
}

type RestoreFilePreview struct {
	FilePath    string `json:"file_path"`
	Action      string `json:"action"` // "restore", "create", "skip"
	Diff        string `json:"diff,omitempty"`
	CurrentHash string `json:"current_hash,omitempty"`
	BackupHash  string `json:"backup_hash"`
}
