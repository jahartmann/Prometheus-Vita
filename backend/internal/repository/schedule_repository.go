package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduleRepository interface {
	Create(ctx context.Context, schedule *model.BackupSchedule) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.BackupSchedule, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.BackupSchedule, error)
	ListDue(ctx context.Context) ([]model.BackupSchedule, error)
	Update(ctx context.Context, schedule *model.BackupSchedule) error
	UpdateNextRun(ctx context.Context, id uuid.UUID, lastRun, nextRun time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgScheduleRepository struct {
	db *pgxpool.Pool
}

func NewScheduleRepository(db *pgxpool.Pool) ScheduleRepository {
	return &pgScheduleRepository{db: db}
}

func (r *pgScheduleRepository) Create(ctx context.Context, schedule *model.BackupSchedule) error {
	schedule.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO backup_schedules (id, node_id, cron_expression, is_active,
		        retention_count, last_run_at, next_run_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
		schedule.ID, schedule.NodeID, schedule.CronExpression, schedule.IsActive,
		schedule.RetentionCount, schedule.LastRunAt, schedule.NextRunAt,
	)
	if err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}
	return nil
}

func (r *pgScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.BackupSchedule, error) {
	var s model.BackupSchedule
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, cron_expression, is_active, retention_count,
		        last_run_at, next_run_at, created_at, updated_at
		 FROM backup_schedules WHERE id = $1`, id,
	).Scan(&s.ID, &s.NodeID, &s.CronExpression, &s.IsActive, &s.RetentionCount,
		&s.LastRunAt, &s.NextRunAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get schedule by id: %w", err)
	}
	return &s, nil
}

func (r *pgScheduleRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.BackupSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, cron_expression, is_active, retention_count,
		        last_run_at, next_run_at, created_at, updated_at
		 FROM backup_schedules WHERE node_id = $1 ORDER BY created_at ASC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list schedules by node: %w", err)
	}
	defer rows.Close()

	var schedules []model.BackupSchedule
	for rows.Next() {
		var s model.BackupSchedule
		if err := rows.Scan(&s.ID, &s.NodeID, &s.CronExpression, &s.IsActive, &s.RetentionCount,
			&s.LastRunAt, &s.NextRunAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func (r *pgScheduleRepository) ListDue(ctx context.Context) ([]model.BackupSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, cron_expression, is_active, retention_count,
		        last_run_at, next_run_at, created_at, updated_at
		 FROM backup_schedules WHERE is_active = true AND next_run_at <= NOW()`)
	if err != nil {
		return nil, fmt.Errorf("list due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []model.BackupSchedule
	for rows.Next() {
		var s model.BackupSchedule
		if err := rows.Scan(&s.ID, &s.NodeID, &s.CronExpression, &s.IsActive, &s.RetentionCount,
			&s.LastRunAt, &s.NextRunAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan due schedule: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func (r *pgScheduleRepository) Update(ctx context.Context, schedule *model.BackupSchedule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE backup_schedules SET cron_expression=$1, is_active=$2, retention_count=$3,
		        next_run_at=$4, updated_at=NOW()
		 WHERE id=$5`,
		schedule.CronExpression, schedule.IsActive, schedule.RetentionCount,
		schedule.NextRunAt, schedule.ID,
	)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	return nil
}

func (r *pgScheduleRepository) UpdateNextRun(ctx context.Context, id uuid.UUID, lastRun, nextRun time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE backup_schedules SET last_run_at=$1, next_run_at=$2, updated_at=NOW()
		 WHERE id=$3`,
		lastRun, nextRun, id,
	)
	if err != nil {
		return fmt.Errorf("update schedule next run: %w", err)
	}
	return nil
}

func (r *pgScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM backup_schedules WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}
