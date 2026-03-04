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

type MigrationRepository interface {
	Create(ctx context.Context, m *model.VMMigration) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.VMMigration, error)
	List(ctx context.Context) ([]model.VMMigration, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.VMMigration, error)
	Update(ctx context.Context, m *model.VMMigration) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgMigrationRepository struct {
	db *pgxpool.Pool
}

func NewMigrationRepository(db *pgxpool.Pool) MigrationRepository {
	return &pgMigrationRepository{db: db}
}

const migrationColumns = `id, source_node_id, target_node_id, vmid, vm_name, vm_type,
	status, mode, target_storage, progress, current_step,
	vzdump_file_path, vzdump_file_size, vzdump_task_upid,
	transfer_bytes_sent, transfer_speed_bps,
	new_vmid, restore_task_upid,
	cleanup_source, cleanup_target, error_message,
	started_at, completed_at, created_at, updated_at, initiated_by`

func scanMigration(row pgx.Row) (*model.VMMigration, error) {
	var m model.VMMigration
	err := row.Scan(
		&m.ID, &m.SourceNodeID, &m.TargetNodeID, &m.VMID, &m.VMName, &m.VMType,
		&m.Status, &m.Mode, &m.TargetStorage, &m.Progress, &m.CurrentStep,
		&m.VzdumpFilePath, &m.VzdumpFileSize, &m.VzdumpTaskUPID,
		&m.TransferBytesSent, &m.TransferSpeedBps,
		&m.NewVMID, &m.RestoreTaskUPID,
		&m.CleanupSource, &m.CleanupTarget, &m.ErrorMessage,
		&m.StartedAt, &m.CompletedAt, &m.CreatedAt, &m.UpdatedAt, &m.InitiatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *pgMigrationRepository) Create(ctx context.Context, m *model.VMMigration) error {
	m.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO vm_migrations (id, source_node_id, target_node_id, vmid, vm_name, vm_type,
			status, mode, target_storage, progress, current_step,
			vzdump_file_path, vzdump_file_size, vzdump_task_upid,
			transfer_bytes_sent, transfer_speed_bps,
			new_vmid, restore_task_upid,
			cleanup_source, cleanup_target, error_message,
			started_at, completed_at, created_at, updated_at, initiated_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,NOW(),NOW(),$24)`,
		m.ID, m.SourceNodeID, m.TargetNodeID, m.VMID, m.VMName, m.VMType,
		m.Status, m.Mode, m.TargetStorage, m.Progress, m.CurrentStep,
		m.VzdumpFilePath, m.VzdumpFileSize, m.VzdumpTaskUPID,
		m.TransferBytesSent, m.TransferSpeedBps,
		m.NewVMID, m.RestoreTaskUPID,
		m.CleanupSource, m.CleanupTarget, m.ErrorMessage,
		m.StartedAt, m.CompletedAt, m.InitiatedBy,
	)
	if err != nil {
		return fmt.Errorf("create migration: %w", err)
	}
	return nil
}

func (r *pgMigrationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.VMMigration, error) {
	m, err := scanMigration(r.db.QueryRow(ctx,
		`SELECT `+migrationColumns+` FROM vm_migrations WHERE id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("get migration by id: %w", err)
	}
	return m, nil
}

func (r *pgMigrationRepository) List(ctx context.Context) ([]model.VMMigration, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+migrationColumns+` FROM vm_migrations ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	defer rows.Close()

	var migrations []model.VMMigration
	for rows.Next() {
		var m model.VMMigration
		if err := rows.Scan(
			&m.ID, &m.SourceNodeID, &m.TargetNodeID, &m.VMID, &m.VMName, &m.VMType,
			&m.Status, &m.Mode, &m.TargetStorage, &m.Progress, &m.CurrentStep,
			&m.VzdumpFilePath, &m.VzdumpFileSize, &m.VzdumpTaskUPID,
			&m.TransferBytesSent, &m.TransferSpeedBps,
			&m.NewVMID, &m.RestoreTaskUPID,
			&m.CleanupSource, &m.CleanupTarget, &m.ErrorMessage,
			&m.StartedAt, &m.CompletedAt, &m.CreatedAt, &m.UpdatedAt, &m.InitiatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan migration: %w", err)
		}
		migrations = append(migrations, m)
	}
	return migrations, rows.Err()
}

func (r *pgMigrationRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.VMMigration, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+migrationColumns+` FROM vm_migrations
		 WHERE source_node_id = $1 OR target_node_id = $1
		 ORDER BY created_at DESC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list migrations by node: %w", err)
	}
	defer rows.Close()

	var migrations []model.VMMigration
	for rows.Next() {
		var m model.VMMigration
		if err := rows.Scan(
			&m.ID, &m.SourceNodeID, &m.TargetNodeID, &m.VMID, &m.VMName, &m.VMType,
			&m.Status, &m.Mode, &m.TargetStorage, &m.Progress, &m.CurrentStep,
			&m.VzdumpFilePath, &m.VzdumpFileSize, &m.VzdumpTaskUPID,
			&m.TransferBytesSent, &m.TransferSpeedBps,
			&m.NewVMID, &m.RestoreTaskUPID,
			&m.CleanupSource, &m.CleanupTarget, &m.ErrorMessage,
			&m.StartedAt, &m.CompletedAt, &m.CreatedAt, &m.UpdatedAt, &m.InitiatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan migration: %w", err)
		}
		migrations = append(migrations, m)
	}
	return migrations, rows.Err()
}

func (r *pgMigrationRepository) Update(ctx context.Context, m *model.VMMigration) error {
	_, err := r.db.Exec(ctx,
		`UPDATE vm_migrations SET
			status=$1, progress=$2, current_step=$3,
			vzdump_file_path=$4, vzdump_file_size=$5, vzdump_task_upid=$6,
			transfer_bytes_sent=$7, transfer_speed_bps=$8,
			new_vmid=$9, restore_task_upid=$10,
			error_message=$11, started_at=$12, completed_at=$13,
			updated_at=NOW()
		 WHERE id=$14`,
		m.Status, m.Progress, m.CurrentStep,
		m.VzdumpFilePath, m.VzdumpFileSize, m.VzdumpTaskUPID,
		m.TransferBytesSent, m.TransferSpeedBps,
		m.NewVMID, m.RestoreTaskUPID,
		m.ErrorMessage, m.StartedAt, m.CompletedAt,
		m.ID,
	)
	if err != nil {
		return fmt.Errorf("update migration: %w", err)
	}
	return nil
}

func (r *pgMigrationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, "DELETE FROM vm_migrations WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete migration: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
