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

type NetworkAnomalyRepository interface {
	Create(ctx context.Context, anomaly *model.NetworkAnomaly) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.NetworkAnomaly, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID, limit, offset int) ([]model.NetworkAnomaly, error)
	Acknowledge(ctx context.Context, id, userID uuid.UUID) error
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type pgNetworkAnomalyRepository struct {
	db *pgxpool.Pool
}

func NewNetworkAnomalyRepository(db *pgxpool.Pool) NetworkAnomalyRepository {
	return &pgNetworkAnomalyRepository{db: db}
}

func (r *pgNetworkAnomalyRepository) Create(ctx context.Context, anomaly *model.NetworkAnomaly) error {
	anomaly.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO network_anomalies (id, node_id, anomaly_type, risk_score, details_json, scan_id, is_acknowledged, acknowledged_at, acknowledged_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
		anomaly.ID, anomaly.NodeID, anomaly.AnomalyType, anomaly.RiskScore, anomaly.DetailsJSON,
		anomaly.ScanID, anomaly.IsAcknowledged, anomaly.AcknowledgedAt, anomaly.AcknowledgedBy,
	)
	if err != nil {
		return fmt.Errorf("create network anomaly: %w", err)
	}
	return nil
}

func (r *pgNetworkAnomalyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.NetworkAnomaly, error) {
	var a model.NetworkAnomaly
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, anomaly_type, risk_score, details_json, scan_id, is_acknowledged, acknowledged_at, acknowledged_by, created_at
		 FROM network_anomalies WHERE id = $1`, id,
	).Scan(&a.ID, &a.NodeID, &a.AnomalyType, &a.RiskScore, &a.DetailsJSON, &a.ScanID,
		&a.IsAcknowledged, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get network anomaly: %w", err)
	}
	return &a, nil
}

func (r *pgNetworkAnomalyRepository) ListByNode(ctx context.Context, nodeID uuid.UUID, limit, offset int) ([]model.NetworkAnomaly, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, anomaly_type, risk_score, details_json, scan_id, is_acknowledged, acknowledged_at, acknowledged_by, created_at
		 FROM network_anomalies WHERE node_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		nodeID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list network anomalies: %w", err)
	}
	defer rows.Close()

	var anomalies []model.NetworkAnomaly
	for rows.Next() {
		var a model.NetworkAnomaly
		if err := rows.Scan(&a.ID, &a.NodeID, &a.AnomalyType, &a.RiskScore, &a.DetailsJSON, &a.ScanID,
			&a.IsAcknowledged, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan network anomaly: %w", err)
		}
		anomalies = append(anomalies, a)
	}
	return anomalies, rows.Err()
}

func (r *pgNetworkAnomalyRepository) Acknowledge(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE network_anomalies SET is_acknowledged=true, acknowledged_at=NOW(), acknowledged_by=$2 WHERE id=$1`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("acknowledge network anomaly: %w", err)
	}
	return nil
}

func (r *pgNetworkAnomalyRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `DELETE FROM network_anomalies WHERE created_at < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("delete old network anomalies: %w", err)
	}
	return result.RowsAffected(), nil
}
