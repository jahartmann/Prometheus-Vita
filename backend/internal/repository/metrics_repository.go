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

type MetricsRepository interface {
	Insert(ctx context.Context, record *model.MetricsRecord) error
	GetByNode(ctx context.Context, nodeID uuid.UUID, since, until time.Time) ([]model.MetricsRecord, error)
	GetLatest(ctx context.Context, nodeID uuid.UUID) (*model.MetricsRecord, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type pgMetricsRepository struct {
	db *pgxpool.Pool
}

func NewMetricsRepository(db *pgxpool.Pool) MetricsRepository {
	return &pgMetricsRepository{db: db}
}

func (r *pgMetricsRepository) Insert(ctx context.Context, record *model.MetricsRecord) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO metrics_records (node_id, recorded_at, cpu_usage, mem_used, mem_total,
		        disk_used, disk_total, net_in, net_out, load_avg)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id`,
		record.NodeID, record.RecordedAt, record.CPUUsage,
		record.MemUsed, record.MemTotal,
		record.DiskUsed, record.DiskTotal,
		record.NetIn, record.NetOut, record.LoadAvg,
	).Scan(&record.ID)
	if err != nil {
		return fmt.Errorf("insert metrics record: %w", err)
	}
	return nil
}

func (r *pgMetricsRepository) GetByNode(ctx context.Context, nodeID uuid.UUID, since, until time.Time) ([]model.MetricsRecord, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, recorded_at, cpu_usage, mem_used, mem_total,
		        disk_used, disk_total, net_in, net_out, load_avg
		 FROM metrics_records
		 WHERE node_id = $1 AND recorded_at BETWEEN $2 AND $3
		 ORDER BY recorded_at ASC`, nodeID, since, until)
	if err != nil {
		return nil, fmt.Errorf("get metrics by node: %w", err)
	}
	defer rows.Close()

	var records []model.MetricsRecord
	for rows.Next() {
		var m model.MetricsRecord
		if err := rows.Scan(&m.ID, &m.NodeID, &m.RecordedAt, &m.CPUUsage,
			&m.MemUsed, &m.MemTotal,
			&m.DiskUsed, &m.DiskTotal,
			&m.NetIn, &m.NetOut, &m.LoadAvg); err != nil {
			return nil, fmt.Errorf("scan metrics record: %w", err)
		}
		records = append(records, m)
	}
	return records, rows.Err()
}

func (r *pgMetricsRepository) GetLatest(ctx context.Context, nodeID uuid.UUID) (*model.MetricsRecord, error) {
	var m model.MetricsRecord
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, recorded_at, cpu_usage, mem_used, mem_total,
		        disk_used, disk_total, net_in, net_out, load_avg
		 FROM metrics_records WHERE node_id = $1 ORDER BY recorded_at DESC LIMIT 1`, nodeID,
	).Scan(&m.ID, &m.NodeID, &m.RecordedAt, &m.CPUUsage,
		&m.MemUsed, &m.MemTotal,
		&m.DiskUsed, &m.DiskTotal,
		&m.NetIn, &m.NetOut, &m.LoadAvg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get latest metrics: %w", err)
	}
	return &m, nil
}

func (r *pgMetricsRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	ct, err := r.db.Exec(ctx, "DELETE FROM metrics_records WHERE recorded_at < $1", before)
	if err != nil {
		return 0, fmt.Errorf("delete old metrics: %w", err)
	}
	return ct.RowsAffected(), nil
}
