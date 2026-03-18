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

type ScanBaselineRepository interface {
	Create(ctx context.Context, baseline *model.ScanBaseline) error
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.ScanBaseline, error)
	GetActive(ctx context.Context, nodeID uuid.UUID) (*model.ScanBaseline, error)
	Activate(ctx context.Context, nodeID, baselineID uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, req model.UpdateBaselineRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgScanBaselineRepository struct {
	db *pgxpool.Pool
}

func NewScanBaselineRepository(db *pgxpool.Pool) ScanBaselineRepository {
	return &pgScanBaselineRepository{db: db}
}

func (r *pgScanBaselineRepository) Create(ctx context.Context, baseline *model.ScanBaseline) error {
	baseline.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO scan_baselines (id, node_id, label, is_active, baseline_json, whitelist_json, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		baseline.ID, baseline.NodeID, baseline.Label, baseline.IsActive, baseline.BaselineJSON, baseline.WhitelistJSON,
	)
	if err != nil {
		return fmt.Errorf("create scan baseline: %w", err)
	}
	return nil
}

func (r *pgScanBaselineRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.ScanBaseline, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, label, is_active, baseline_json, whitelist_json, created_at
		 FROM scan_baselines WHERE node_id = $1 ORDER BY created_at DESC`,
		nodeID)
	if err != nil {
		return nil, fmt.Errorf("list scan baselines: %w", err)
	}
	defer rows.Close()

	var baselines []model.ScanBaseline
	for rows.Next() {
		var b model.ScanBaseline
		if err := rows.Scan(&b.ID, &b.NodeID, &b.Label, &b.IsActive, &b.BaselineJSON, &b.WhitelistJSON, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan scan baseline: %w", err)
		}
		baselines = append(baselines, b)
	}
	return baselines, rows.Err()
}

func (r *pgScanBaselineRepository) GetActive(ctx context.Context, nodeID uuid.UUID) (*model.ScanBaseline, error) {
	var b model.ScanBaseline
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, label, is_active, baseline_json, whitelist_json, created_at
		 FROM scan_baselines WHERE node_id=$1 AND is_active=true`, nodeID,
	).Scan(&b.ID, &b.NodeID, &b.Label, &b.IsActive, &b.BaselineJSON, &b.WhitelistJSON, &b.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get active scan baseline: %w", err)
	}
	return &b, nil
}

func (r *pgScanBaselineRepository) Activate(ctx context.Context, nodeID, baselineID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx for activate baseline: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE scan_baselines SET is_active=false WHERE node_id=$1`, nodeID)
	if err != nil {
		return fmt.Errorf("deactivate scan baselines: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE scan_baselines SET is_active=true WHERE id=$1`, baselineID)
	if err != nil {
		return fmt.Errorf("activate scan baseline: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit activate baseline: %w", err)
	}
	return nil
}

func (r *pgScanBaselineRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateBaselineRequest) error {
	if req.Label != nil {
		_, err := r.db.Exec(ctx, `UPDATE scan_baselines SET label=$2 WHERE id=$1`, id, *req.Label)
		if err != nil {
			return fmt.Errorf("update scan baseline label: %w", err)
		}
	}
	if req.WhitelistJSON != nil {
		_, err := r.db.Exec(ctx, `UPDATE scan_baselines SET whitelist_json=$2 WHERE id=$1`, id, *req.WhitelistJSON)
		if err != nil {
			return fmt.Errorf("update scan baseline whitelist: %w", err)
		}
	}
	return nil
}

func (r *pgScanBaselineRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM scan_baselines WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete scan baseline: %w", err)
	}
	return nil
}
