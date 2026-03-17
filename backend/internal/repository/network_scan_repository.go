package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NetworkScanRepository interface {
	Create(ctx context.Context, scan *model.NetworkScan) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.NetworkScan, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID, limit, offset int) ([]model.NetworkScan, error)
	Complete(ctx context.Context, id uuid.UUID, resultsJSON json.RawMessage) error
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type pgNetworkScanRepository struct {
	db *pgxpool.Pool
}

func NewNetworkScanRepository(db *pgxpool.Pool) NetworkScanRepository {
	return &pgNetworkScanRepository{db: db}
}

func (r *pgNetworkScanRepository) Create(ctx context.Context, scan *model.NetworkScan) error {
	scan.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO network_scans (id, node_id, scan_type, results_json, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, NOW(), $5)`,
		scan.ID, scan.NodeID, scan.ScanType, scan.ResultsJSON, scan.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("create network scan: %w", err)
	}
	return nil
}

func (r *pgNetworkScanRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.NetworkScan, error) {
	var s model.NetworkScan
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, scan_type, results_json, started_at, completed_at
		 FROM network_scans WHERE id = $1`, id,
	).Scan(&s.ID, &s.NodeID, &s.ScanType, &s.ResultsJSON, &s.StartedAt, &s.CompletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get network scan: %w", err)
	}
	return &s, nil
}

func (r *pgNetworkScanRepository) ListByNode(ctx context.Context, nodeID uuid.UUID, limit, offset int) ([]model.NetworkScan, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, scan_type, results_json, started_at, completed_at
		 FROM network_scans WHERE node_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3`,
		nodeID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list network scans: %w", err)
	}
	defer rows.Close()

	var scans []model.NetworkScan
	for rows.Next() {
		var s model.NetworkScan
		if err := rows.Scan(&s.ID, &s.NodeID, &s.ScanType, &s.ResultsJSON, &s.StartedAt, &s.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan network scan: %w", err)
		}
		scans = append(scans, s)
	}
	return scans, rows.Err()
}

func (r *pgNetworkScanRepository) Complete(ctx context.Context, id uuid.UUID, resultsJSON json.RawMessage) error {
	_, err := r.db.Exec(ctx,
		`UPDATE network_scans SET results_json=$2, completed_at=NOW() WHERE id=$1`,
		id, resultsJSON,
	)
	if err != nil {
		return fmt.Errorf("complete network scan: %w", err)
	}
	return nil
}

func (r *pgNetworkScanRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `DELETE FROM network_scans WHERE started_at < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("delete old network scans: %w", err)
	}
	return result.RowsAffected(), nil
}
