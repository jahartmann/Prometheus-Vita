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

type AnomalyRepository interface {
	Create(ctx context.Context, record *model.AnomalyRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.AnomalyRecord, error)
	ListUnresolved(ctx context.Context) ([]model.AnomalyRecord, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.AnomalyRecord, error)
	ListFiltered(ctx context.Context, filter model.QueryFilter) ([]model.AnomalyRecord, error)
	Resolve(ctx context.Context, id uuid.UUID) error
}

type pgAnomalyRepository struct {
	db *pgxpool.Pool
}

func NewAnomalyRepository(db *pgxpool.Pool) AnomalyRepository {
	return &pgAnomalyRepository{db: db}
}

func (r *pgAnomalyRepository) Create(ctx context.Context, record *model.AnomalyRecord) error {
	record.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO anomaly_records (id, node_id, metric, value, z_score, mean, stddev, severity, is_resolved, detected_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
		record.ID, record.NodeID, record.Metric, record.Value, record.ZScore,
		record.Mean, record.StdDev, record.Severity, record.IsResolved,
	)
	if err != nil {
		return fmt.Errorf("create anomaly record: %w", err)
	}
	return nil
}

func (r *pgAnomalyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AnomalyRecord, error) {
	var a model.AnomalyRecord
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, metric, value, z_score, mean, stddev, severity, is_resolved, detected_at, resolved_at
		 FROM anomaly_records WHERE id = $1`, id,
	).Scan(&a.ID, &a.NodeID, &a.Metric, &a.Value, &a.ZScore, &a.Mean, &a.StdDev,
		&a.Severity, &a.IsResolved, &a.DetectedAt, &a.ResolvedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get anomaly by id: %w", err)
	}
	return &a, nil
}

func (r *pgAnomalyRepository) ListUnresolved(ctx context.Context) ([]model.AnomalyRecord, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, metric, value, z_score, mean, stddev, severity, is_resolved, detected_at, resolved_at
		 FROM anomaly_records WHERE is_resolved = false ORDER BY detected_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list unresolved anomalies: %w", err)
	}
	defer rows.Close()

	var records []model.AnomalyRecord
	for rows.Next() {
		var a model.AnomalyRecord
		if err := rows.Scan(&a.ID, &a.NodeID, &a.Metric, &a.Value, &a.ZScore, &a.Mean, &a.StdDev,
			&a.Severity, &a.IsResolved, &a.DetectedAt, &a.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scan anomaly: %w", err)
		}
		records = append(records, a)
	}
	return records, rows.Err()
}

func (r *pgAnomalyRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.AnomalyRecord, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, metric, value, z_score, mean, stddev, severity, is_resolved, detected_at, resolved_at
		 FROM anomaly_records WHERE node_id = $1 ORDER BY detected_at DESC LIMIT 100`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list anomalies by node: %w", err)
	}
	defer rows.Close()

	var records []model.AnomalyRecord
	for rows.Next() {
		var a model.AnomalyRecord
		if err := rows.Scan(&a.ID, &a.NodeID, &a.Metric, &a.Value, &a.ZScore, &a.Mean, &a.StdDev,
			&a.Severity, &a.IsResolved, &a.DetectedAt, &a.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scan anomaly: %w", err)
		}
		records = append(records, a)
	}
	return records, rows.Err()
}

func (r *pgAnomalyRepository) ListFiltered(ctx context.Context, filter model.QueryFilter) ([]model.AnomalyRecord, error) {
	limit := filter.NormalizedLimit(100, 500)
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, metric, value, z_score, mean, stddev, severity, is_resolved, detected_at, resolved_at
		 FROM anomaly_records
		 WHERE ($1::timestamptz IS NULL OR detected_at >= $1)
		   AND ($2::timestamptz IS NULL OR detected_at <= $2)
		   AND ($3::uuid IS NULL OR node_id = $3)
		   AND ($4 = '' OR $4 = 'all' OR severity = $4)
		   AND ($5 = '' OR $5 = 'all' OR ($5 = 'resolved' AND is_resolved = true) OR ($5 = 'open' AND is_resolved = false) OR ($5 = 'unresolved' AND is_resolved = false))
		   AND ($6 = '' OR metric ILIKE '%' || $6 || '%')
		 ORDER BY detected_at DESC
		 LIMIT $7 OFFSET $8`,
		filter.From, filter.To, filter.NodeID, filter.Severity, filter.Status, filter.Query, limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("list filtered anomalies: %w", err)
	}
	defer rows.Close()

	var records []model.AnomalyRecord
	for rows.Next() {
		var a model.AnomalyRecord
		if err := rows.Scan(&a.ID, &a.NodeID, &a.Metric, &a.Value, &a.ZScore, &a.Mean, &a.StdDev,
			&a.Severity, &a.IsResolved, &a.DetectedAt, &a.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scan filtered anomaly: %w", err)
		}
		records = append(records, a)
	}
	return records, rows.Err()
}

func (r *pgAnomalyRepository) Resolve(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE anomaly_records SET is_resolved = true, resolved_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("resolve anomaly: %w", err)
	}
	return nil
}
