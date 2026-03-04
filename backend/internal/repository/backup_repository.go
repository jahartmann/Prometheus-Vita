package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BackupRepository interface {
	Create(ctx context.Context, backup *model.ConfigBackup) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.ConfigBackup, error)
	ListAll(ctx context.Context) ([]model.ConfigBackup, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.ConfigBackup, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.BackupStatus, errorMsg string) error
	UpdateCompleted(ctx context.Context, id uuid.UUID, fileCount int, totalSize int64) error
	UpdateRecoveryGuide(ctx context.Context, id uuid.UUID, guide string) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetLatestByNode(ctx context.Context, nodeID uuid.UUID) (*model.ConfigBackup, error)
	GetNextVersion(ctx context.Context, nodeID uuid.UUID) (int, error)
	CountByNode(ctx context.Context, nodeID uuid.UUID) (int, error)
	DeleteOldest(ctx context.Context, nodeID uuid.UUID, keepCount int) error
}

type pgBackupRepository struct {
	db *pgxpool.Pool
}

func NewBackupRepository(db *pgxpool.Pool) BackupRepository {
	return &pgBackupRepository{db: db}
}

func (r *pgBackupRepository) Create(ctx context.Context, backup *model.ConfigBackup) error {
	backup.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO config_backups (id, node_id, version, backup_type, file_count, total_size,
		        status, error_message, notes, created_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $10)`,
		backup.ID, backup.NodeID, backup.Version, backup.BackupType,
		backup.FileCount, backup.TotalSize, backup.Status,
		backup.ErrorMessage, backup.Notes, backup.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("create backup: %w", err)
	}
	return nil
}

func (r *pgBackupRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ConfigBackup, error) {
	var b model.ConfigBackup
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, version, backup_type, file_count, total_size,
		        status, error_message, notes, recovery_guide, created_at, completed_at
		 FROM config_backups WHERE id = $1`, id,
	).Scan(&b.ID, &b.NodeID, &b.Version, &b.BackupType,
		&b.FileCount, &b.TotalSize, &b.Status,
		&b.ErrorMessage, &b.Notes, &b.RecoveryGuide, &b.CreatedAt, &b.CompletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get backup by id: %w", err)
	}
	return &b, nil
}

func (r *pgBackupRepository) ListAll(ctx context.Context) ([]model.ConfigBackup, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, version, backup_type, file_count, total_size,
		        status, error_message, notes, recovery_guide, created_at, completed_at
		 FROM config_backups ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list all backups: %w", err)
	}
	defer rows.Close()

	var backups []model.ConfigBackup
	for rows.Next() {
		var b model.ConfigBackup
		if err := rows.Scan(&b.ID, &b.NodeID, &b.Version, &b.BackupType,
			&b.FileCount, &b.TotalSize, &b.Status,
			&b.ErrorMessage, &b.Notes, &b.CreatedAt, &b.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		backups = append(backups, b)
	}
	return backups, rows.Err()
}

func (r *pgBackupRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.ConfigBackup, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, version, backup_type, file_count, total_size,
		        status, error_message, notes, recovery_guide, created_at, completed_at
		 FROM config_backups WHERE node_id = $1 ORDER BY created_at DESC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list backups by node: %w", err)
	}
	defer rows.Close()

	var backups []model.ConfigBackup
	for rows.Next() {
		var b model.ConfigBackup
		if err := rows.Scan(&b.ID, &b.NodeID, &b.Version, &b.BackupType,
			&b.FileCount, &b.TotalSize, &b.Status,
			&b.ErrorMessage, &b.Notes, &b.CreatedAt, &b.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		backups = append(backups, b)
	}
	return backups, rows.Err()
}

func (r *pgBackupRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.BackupStatus, errorMsg string) error {
	var err error
	if status == model.BackupStatusCompleted || status == model.BackupStatusFailed {
		_, err = r.db.Exec(ctx,
			`UPDATE config_backups SET status=$1, error_message=$2, completed_at=NOW() WHERE id=$3`,
			status, errorMsg, id,
		)
	} else {
		_, err = r.db.Exec(ctx,
			`UPDATE config_backups SET status=$1, error_message=$2 WHERE id=$3`,
			status, errorMsg, id,
		)
	}
	if err != nil {
		return fmt.Errorf("update backup status: %w", err)
	}
	return nil
}

func (r *pgBackupRepository) UpdateCompleted(ctx context.Context, id uuid.UUID, fileCount int, totalSize int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE config_backups SET file_count=$1, total_size=$2, status=$3, completed_at=NOW()
		 WHERE id=$4`,
		fileCount, totalSize, model.BackupStatusCompleted, id,
	)
	if err != nil {
		return fmt.Errorf("update backup completed: %w", err)
	}
	return nil
}

func (r *pgBackupRepository) UpdateRecoveryGuide(ctx context.Context, id uuid.UUID, guide string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE config_backups SET recovery_guide=$1 WHERE id=$2`,
		guide, id,
	)
	if err != nil {
		return fmt.Errorf("update recovery guide: %w", err)
	}
	return nil
}

func (r *pgBackupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM config_backups WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete backup: %w", err)
	}
	return nil
}

func (r *pgBackupRepository) GetLatestByNode(ctx context.Context, nodeID uuid.UUID) (*model.ConfigBackup, error) {
	var b model.ConfigBackup
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, version, backup_type, file_count, total_size,
		        status, error_message, notes, recovery_guide, created_at, completed_at
		 FROM config_backups WHERE node_id = $1 ORDER BY created_at DESC LIMIT 1`, nodeID,
	).Scan(&b.ID, &b.NodeID, &b.Version, &b.BackupType,
		&b.FileCount, &b.TotalSize, &b.Status,
		&b.ErrorMessage, &b.Notes, &b.RecoveryGuide, &b.CreatedAt, &b.CompletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get latest backup by node: %w", err)
	}
	return &b, nil
}

func (r *pgBackupRepository) GetNextVersion(ctx context.Context, nodeID uuid.UUID) (int, error) {
	var version int
	err := r.db.QueryRow(ctx,
		"SELECT COALESCE(MAX(version), 0) + 1 FROM config_backups WHERE node_id = $1", nodeID,
	).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("get next backup version: %w", err)
	}
	return version, nil
}

func (r *pgBackupRepository) CountByNode(ctx context.Context, nodeID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM config_backups WHERE node_id = $1", nodeID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count backups by node: %w", err)
	}
	return count, nil
}

func (r *pgBackupRepository) DeleteOldest(ctx context.Context, nodeID uuid.UUID, keepCount int) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM config_backups WHERE id IN (
			SELECT id FROM config_backups WHERE node_id = $1
			ORDER BY created_at DESC OFFSET $2
		)`, nodeID, keepCount,
	)
	if err != nil {
		return fmt.Errorf("delete oldest backups: %w", err)
	}
	return nil
}
