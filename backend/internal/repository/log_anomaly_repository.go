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

type LogAnomalyRepository interface {
	Create(ctx context.Context, anomaly *model.LogAnomaly) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.LogAnomaly, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID, limit, offset int) ([]model.LogAnomaly, error)
	ListFiltered(ctx context.Context, filter model.QueryFilter) ([]model.LogAnomaly, error)
	Acknowledge(ctx context.Context, id, userID uuid.UUID) error
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type pgLogAnomalyRepository struct {
	db *pgxpool.Pool
}

func NewLogAnomalyRepository(db *pgxpool.Pool) LogAnomalyRepository {
	return &pgLogAnomalyRepository{db: db}
}

func (r *pgLogAnomalyRepository) Create(ctx context.Context, anomaly *model.LogAnomaly) error {
	anomaly.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO log_anomalies (id, node_id, timestamp, source, severity, anomaly_score, category, summary, raw_log, is_acknowledged, acknowledged_at, acknowledged_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())`,
		anomaly.ID, anomaly.NodeID, anomaly.Timestamp, anomaly.Source, anomaly.Severity,
		anomaly.AnomalyScore, anomaly.Category, anomaly.Summary, anomaly.RawLog,
		anomaly.IsAcknowledged, anomaly.AcknowledgedAt, anomaly.AcknowledgedBy,
	)
	if err != nil {
		return fmt.Errorf("create log anomaly: %w", err)
	}
	return nil
}

func (r *pgLogAnomalyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.LogAnomaly, error) {
	var a model.LogAnomaly
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, timestamp, source, severity, anomaly_score, category, summary, raw_log, is_acknowledged, acknowledged_at, acknowledged_by, created_at
		 FROM log_anomalies WHERE id = $1`, id,
	).Scan(&a.ID, &a.NodeID, &a.Timestamp, &a.Source, &a.Severity, &a.AnomalyScore,
		&a.Category, &a.Summary, &a.RawLog, &a.IsAcknowledged, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get log anomaly: %w", err)
	}
	return &a, nil
}

func (r *pgLogAnomalyRepository) ListByNode(ctx context.Context, nodeID uuid.UUID, limit, offset int) ([]model.LogAnomaly, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, timestamp, source, severity, anomaly_score, category, summary, raw_log, is_acknowledged, acknowledged_at, acknowledged_by, created_at
		 FROM log_anomalies WHERE node_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3`,
		nodeID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list log anomalies: %w", err)
	}
	defer rows.Close()

	var anomalies []model.LogAnomaly
	for rows.Next() {
		var a model.LogAnomaly
		if err := rows.Scan(&a.ID, &a.NodeID, &a.Timestamp, &a.Source, &a.Severity, &a.AnomalyScore,
			&a.Category, &a.Summary, &a.RawLog, &a.IsAcknowledged, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan log anomaly: %w", err)
		}
		anomalies = append(anomalies, a)
	}
	return anomalies, rows.Err()
}

func (r *pgLogAnomalyRepository) ListFiltered(ctx context.Context, filter model.QueryFilter) ([]model.LogAnomaly, error) {
	limit := filter.NormalizedLimit(100, 500)
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, timestamp, source, severity, anomaly_score, category, summary, raw_log, is_acknowledged, acknowledged_at, acknowledged_by, created_at
		 FROM log_anomalies
		 WHERE ($1::timestamptz IS NULL OR timestamp >= $1)
		   AND ($2::timestamptz IS NULL OR timestamp <= $2)
		   AND ($3::uuid IS NULL OR node_id = $3)
		   AND ($4 = '' OR $4 = 'all' OR severity = $4)
		   AND ($5 = '' OR $5 = 'all' OR category = $5)
		   AND ($6 = '' OR $6 = 'all' OR source = $6)
		   AND ($7 = '' OR $7 = 'all' OR ($7 = 'acknowledged' AND is_acknowledged = true) OR ($7 = 'open' AND is_acknowledged = false) OR ($7 = 'unacknowledged' AND is_acknowledged = false))
		   AND ($8 = '' OR summary ILIKE '%' || $8 || '%' OR raw_log ILIKE '%' || $8 || '%')
		 ORDER BY timestamp DESC
		 LIMIT $9 OFFSET $10`,
		filter.From, filter.To, filter.NodeID, filter.Severity, filter.Category, filter.Source, filter.Status, filter.Query, limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("list filtered log anomalies: %w", err)
	}
	defer rows.Close()

	var anomalies []model.LogAnomaly
	for rows.Next() {
		var a model.LogAnomaly
		if err := rows.Scan(&a.ID, &a.NodeID, &a.Timestamp, &a.Source, &a.Severity, &a.AnomalyScore,
			&a.Category, &a.Summary, &a.RawLog, &a.IsAcknowledged, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan filtered log anomaly: %w", err)
		}
		anomalies = append(anomalies, a)
	}
	return anomalies, rows.Err()
}

func (r *pgLogAnomalyRepository) Acknowledge(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE log_anomalies SET is_acknowledged=true, acknowledged_at=NOW(), acknowledged_by=$2 WHERE id=$1`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("acknowledge log anomaly: %w", err)
	}
	return nil
}

func (r *pgLogAnomalyRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `DELETE FROM log_anomalies WHERE created_at < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("delete old log anomalies: %w", err)
	}
	return result.RowsAffected(), nil
}
