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

type LogReportScheduleRepository interface {
	Create(ctx context.Context, schedule *model.LogReportSchedule) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.LogReportSchedule, error)
	List(ctx context.Context) ([]model.LogReportSchedule, error)
	Update(ctx context.Context, schedule *model.LogReportSchedule) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetDue(ctx context.Context, now time.Time) ([]model.LogReportSchedule, error)
}

type pgLogReportScheduleRepository struct {
	db *pgxpool.Pool
}

func NewLogReportScheduleRepository(db *pgxpool.Pool) LogReportScheduleRepository {
	return &pgLogReportScheduleRepository{db: db}
}

func (r *pgLogReportScheduleRepository) Create(ctx context.Context, schedule *model.LogReportSchedule) error {
	schedule.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO log_report_schedules (id, cron_expression, node_ids, time_window_hours, delivery_channel_ids, is_active, last_run_at, next_run_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		schedule.ID, schedule.CronExpression, schedule.NodeIDs, schedule.TimeWindowHours,
		schedule.DeliveryChannelIDs, schedule.IsActive, schedule.LastRunAt, schedule.NextRunAt,
	)
	if err != nil {
		return fmt.Errorf("create log report schedule: %w", err)
	}
	return nil
}

func (r *pgLogReportScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.LogReportSchedule, error) {
	var s model.LogReportSchedule
	err := r.db.QueryRow(ctx,
		`SELECT id, cron_expression, node_ids, time_window_hours, delivery_channel_ids, is_active, last_run_at, next_run_at, created_at
		 FROM log_report_schedules WHERE id = $1`, id,
	).Scan(&s.ID, &s.CronExpression, &s.NodeIDs, &s.TimeWindowHours, &s.DeliveryChannelIDs,
		&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get log report schedule: %w", err)
	}
	return &s, nil
}

func (r *pgLogReportScheduleRepository) List(ctx context.Context) ([]model.LogReportSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, cron_expression, node_ids, time_window_hours, delivery_channel_ids, is_active, last_run_at, next_run_at, created_at
		 FROM log_report_schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list log report schedules: %w", err)
	}
	defer rows.Close()

	var schedules []model.LogReportSchedule
	for rows.Next() {
		var s model.LogReportSchedule
		if err := rows.Scan(&s.ID, &s.CronExpression, &s.NodeIDs, &s.TimeWindowHours, &s.DeliveryChannelIDs,
			&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan log report schedule: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func (r *pgLogReportScheduleRepository) Update(ctx context.Context, schedule *model.LogReportSchedule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE log_report_schedules SET cron_expression=$2, node_ids=$3, time_window_hours=$4, delivery_channel_ids=$5, is_active=$6, last_run_at=$7, next_run_at=$8 WHERE id=$1`,
		schedule.ID, schedule.CronExpression, schedule.NodeIDs, schedule.TimeWindowHours,
		schedule.DeliveryChannelIDs, schedule.IsActive, schedule.LastRunAt, schedule.NextRunAt,
	)
	if err != nil {
		return fmt.Errorf("update log report schedule: %w", err)
	}
	return nil
}

func (r *pgLogReportScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM log_report_schedules WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete log report schedule: %w", err)
	}
	return nil
}

func (r *pgLogReportScheduleRepository) GetDue(ctx context.Context, now time.Time) ([]model.LogReportSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, cron_expression, node_ids, time_window_hours, delivery_channel_ids, is_active, last_run_at, next_run_at, created_at
		 FROM log_report_schedules WHERE is_active=true AND (next_run_at IS NULL OR next_run_at <= $1)`,
		now)
	if err != nil {
		return nil, fmt.Errorf("get due log report schedules: %w", err)
	}
	defer rows.Close()

	var schedules []model.LogReportSchedule
	for rows.Next() {
		var s model.LogReportSchedule
		if err := rows.Scan(&s.ID, &s.CronExpression, &s.NodeIDs, &s.TimeWindowHours, &s.DeliveryChannelIDs,
			&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan due log report schedule: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}
