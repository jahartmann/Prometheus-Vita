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

type DriftRepository interface {
	Create(ctx context.Context, check *model.DriftCheck) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.DriftCheck, error)
	GetLatestByNode(ctx context.Context, nodeID uuid.UUID) (*model.DriftCheck, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.DriftCheck, error)
	ListAll(ctx context.Context, limit int) ([]model.DriftCheck, error)
	Update(ctx context.Context, check *model.DriftCheck) error
}

type pgDriftRepository struct {
	db *pgxpool.Pool
}

func NewDriftRepository(db *pgxpool.Pool) DriftRepository {
	return &pgDriftRepository{db: db}
}

func (r *pgDriftRepository) Create(ctx context.Context, check *model.DriftCheck) error {
	check.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO drift_checks (id, node_id, status, total_files, changed_files, added_files, removed_files, details, ai_analysis, error_message, baseline_updated_at, checked_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())`,
		check.ID, check.NodeID, check.Status, check.TotalFiles, check.ChangedFiles,
		check.AddedFiles, check.RemovedFiles, check.Details, check.AIAnalysis, check.ErrorMessage, check.BaselineUpdatedAt, check.CheckedAt,
	)
	if err != nil {
		return fmt.Errorf("create drift check: %w", err)
	}
	return nil
}

const driftSelectCols = `id, node_id, status, total_files, changed_files, added_files, removed_files, details, COALESCE(ai_analysis, 'null'::jsonb), COALESCE(error_message, ''), baseline_updated_at, checked_at, created_at`

func (r *pgDriftRepository) scanDriftCheck(row pgx.Row) (*model.DriftCheck, error) {
	var c model.DriftCheck
	err := row.Scan(&c.ID, &c.NodeID, &c.Status, &c.TotalFiles, &c.ChangedFiles,
		&c.AddedFiles, &c.RemovedFiles, &c.Details, &c.AIAnalysis, &c.ErrorMessage, &c.BaselineUpdatedAt, &c.CheckedAt, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *pgDriftRepository) scanDriftChecks(rows pgx.Rows) ([]model.DriftCheck, error) {
	var checks []model.DriftCheck
	for rows.Next() {
		var c model.DriftCheck
		if err := rows.Scan(&c.ID, &c.NodeID, &c.Status, &c.TotalFiles, &c.ChangedFiles,
			&c.AddedFiles, &c.RemovedFiles, &c.Details, &c.AIAnalysis, &c.ErrorMessage, &c.BaselineUpdatedAt, &c.CheckedAt, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan drift check: %w", err)
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (r *pgDriftRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.DriftCheck, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+driftSelectCols+` FROM drift_checks WHERE id = $1`, id)
	c, err := r.scanDriftCheck(row)
	if err != nil {
		return nil, fmt.Errorf("get drift check: %w", err)
	}
	return c, nil
}

func (r *pgDriftRepository) GetLatestByNode(ctx context.Context, nodeID uuid.UUID) (*model.DriftCheck, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+driftSelectCols+` FROM drift_checks WHERE node_id = $1 ORDER BY checked_at DESC LIMIT 1`, nodeID)
	c, err := r.scanDriftCheck(row)
	if err != nil {
		return nil, fmt.Errorf("get latest drift check: %w", err)
	}
	return c, nil
}

func (r *pgDriftRepository) ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.DriftCheck, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+driftSelectCols+` FROM drift_checks WHERE node_id = $1 ORDER BY checked_at DESC LIMIT $2`, nodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list drift checks by node: %w", err)
	}
	defer rows.Close()
	return r.scanDriftChecks(rows)
}

func (r *pgDriftRepository) ListAll(ctx context.Context, limit int) ([]model.DriftCheck, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+driftSelectCols+` FROM drift_checks ORDER BY checked_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list all drift checks: %w", err)
	}
	defer rows.Close()
	return r.scanDriftChecks(rows)
}

func (r *pgDriftRepository) Update(ctx context.Context, check *model.DriftCheck) error {
	_, err := r.db.Exec(ctx,
		`UPDATE drift_checks SET status=$1, total_files=$2, changed_files=$3, added_files=$4, removed_files=$5, details=$6, ai_analysis=$7, error_message=$8, baseline_updated_at=$9, checked_at=$10
		 WHERE id=$11`,
		check.Status, check.TotalFiles, check.ChangedFiles, check.AddedFiles, check.RemovedFiles,
		check.Details, check.AIAnalysis, check.ErrorMessage, check.BaselineUpdatedAt, check.CheckedAt, check.ID,
	)
	if err != nil {
		return fmt.Errorf("update drift check: %w", err)
	}
	return nil
}
