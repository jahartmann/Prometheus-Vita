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


type LogAnalysisRepository interface {
	Create(ctx context.Context, analysis *model.LogAnalysis) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.LogAnalysis, error)
	ListByNodes(ctx context.Context, nodeIDs []uuid.UUID, limit, offset int) ([]model.LogAnalysis, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type pgLogAnalysisRepository struct {
	db *pgxpool.Pool
}

func NewLogAnalysisRepository(db *pgxpool.Pool) LogAnalysisRepository {
	return &pgLogAnalysisRepository{db: db}
}

func (r *pgLogAnalysisRepository) Create(ctx context.Context, analysis *model.LogAnalysis) error {
	analysis.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO log_analyses (id, node_ids, time_from, time_to, report_json, model_used, schedule_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		analysis.ID, analysis.NodeIDs, analysis.TimeFrom, analysis.TimeTo,
		analysis.ReportJSON, analysis.ModelUsed, analysis.ScheduleID,
	)
	if err != nil {
		return fmt.Errorf("create log analysis: %w", err)
	}
	return nil
}

func (r *pgLogAnalysisRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.LogAnalysis, error) {
	var a model.LogAnalysis
	err := r.db.QueryRow(ctx,
		`SELECT id, node_ids, time_from, time_to, report_json, model_used, schedule_id, created_at
		 FROM log_analyses WHERE id = $1`, id,
	).Scan(&a.ID, &a.NodeIDs, &a.TimeFrom, &a.TimeTo, &a.ReportJSON, &a.ModelUsed, &a.ScheduleID, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get log analysis: %w", err)
	}
	return &a, nil
}

func (r *pgLogAnalysisRepository) ListByNodes(ctx context.Context, nodeIDs []uuid.UUID, limit, offset int) ([]model.LogAnalysis, error) {
	var (
		rows pgx.Rows
		err  error
	)

	if len(nodeIDs) == 0 {
		rows, err = r.db.Query(ctx,
			`SELECT id, node_ids, time_from, time_to, report_json, model_used, schedule_id, created_at
			 FROM log_analyses ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
			limit, offset)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id, node_ids, time_from, time_to, report_json, model_used, schedule_id, created_at
			 FROM log_analyses WHERE node_ids && $1::uuid[] ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			nodeIDs, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("list log analyses: %w", err)
	}
	defer rows.Close()

	var analyses []model.LogAnalysis
	for rows.Next() {
		var a model.LogAnalysis
		if err := rows.Scan(&a.ID, &a.NodeIDs, &a.TimeFrom, &a.TimeTo, &a.ReportJSON, &a.ModelUsed, &a.ScheduleID, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan log analysis: %w", err)
		}
		analyses = append(analyses, a)
	}
	return analyses, rows.Err()
}

func (r *pgLogAnalysisRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `DELETE FROM log_analyses WHERE created_at < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("delete old log analyses: %w", err)
	}
	return result.RowsAffected(), nil
}
