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
	GetAllMetrics(ctx context.Context, since, until time.Time) ([]model.MetricsRecord, error)
	GetLatest(ctx context.Context, nodeID uuid.UUID) (*model.MetricsRecord, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)

	// VM metrics
	InsertVMMetrics(ctx context.Context, record *model.VMMetricsRecord) error
	GetVMMetricsHistory(ctx context.Context, nodeID uuid.UUID, vmid int, start, end time.Time) ([]model.VMMetricsRecord, error)
	GetVMNetworkSummary(ctx context.Context, nodeID uuid.UUID, vmid int, start, end time.Time) (*model.NetworkSummary, error)
	GetNodeNetworkSummary(ctx context.Context, nodeID uuid.UUID, start, end time.Time) (*model.NetworkSummary, error)
	GetClusterNetworkSummary(ctx context.Context, start, end time.Time) (*model.NetworkSummary, error)
	CleanupVMMetrics(ctx context.Context, before time.Time) (int64, error)
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

func (r *pgMetricsRepository) GetAllMetrics(ctx context.Context, since, until time.Time) ([]model.MetricsRecord, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, recorded_at, cpu_usage, mem_used, mem_total,
		        disk_used, disk_total, net_in, net_out, load_avg
		 FROM metrics_records
		 WHERE recorded_at BETWEEN $1 AND $2
		 ORDER BY recorded_at ASC`, since, until)
	if err != nil {
		return nil, fmt.Errorf("get all metrics: %w", err)
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

func (r *pgMetricsRepository) InsertVMMetrics(ctx context.Context, record *model.VMMetricsRecord) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO vm_metrics_history (node_id, vmid, vm_type, cpu_usage, mem_used, mem_total,
		        net_in, net_out, disk_read, disk_write, recorded_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		record.NodeID, record.VMID, record.VMType, record.CPUUsage,
		record.MemUsed, record.MemTotal,
		record.NetIn, record.NetOut,
		record.DiskRead, record.DiskWrite, record.RecordedAt,
	)
	if err != nil {
		return fmt.Errorf("insert vm metrics: %w", err)
	}
	return nil
}

func (r *pgMetricsRepository) GetVMMetricsHistory(ctx context.Context, nodeID uuid.UUID, vmid int, start, end time.Time) ([]model.VMMetricsRecord, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, vmid, vm_type, cpu_usage, mem_used, mem_total,
		        net_in, net_out, disk_read, disk_write, recorded_at
		 FROM vm_metrics_history
		 WHERE node_id = $1 AND vmid = $2 AND recorded_at BETWEEN $3 AND $4
		 ORDER BY recorded_at ASC`, nodeID, vmid, start, end)
	if err != nil {
		return nil, fmt.Errorf("get vm metrics history: %w", err)
	}
	defer rows.Close()

	var records []model.VMMetricsRecord
	for rows.Next() {
		var m model.VMMetricsRecord
		if err := rows.Scan(&m.ID, &m.NodeID, &m.VMID, &m.VMType, &m.CPUUsage,
			&m.MemUsed, &m.MemTotal,
			&m.NetIn, &m.NetOut,
			&m.DiskRead, &m.DiskWrite, &m.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan vm metrics record: %w", err)
		}
		records = append(records, m)
	}
	return records, rows.Err()
}

func (r *pgMetricsRepository) GetVMNetworkSummary(ctx context.Context, nodeID uuid.UUID, vmid int, start, end time.Time) (*model.NetworkSummary, error) {
	var s model.NetworkSummary
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(net_in * 60), 0), COALESCE(SUM(net_out * 60), 0),
		        COALESCE(AVG(net_in), 0)::bigint, COALESCE(AVG(net_out), 0)::bigint,
		        COALESCE(MAX(net_in), 0), COALESCE(MAX(net_out), 0)
		 FROM vm_metrics_history
		 WHERE node_id = $1 AND vmid = $2 AND recorded_at BETWEEN $3 AND $4`,
		nodeID, vmid, start, end,
	).Scan(&s.TotalIn, &s.TotalOut, &s.AvgInRate, &s.AvgOutRate, &s.PeakInRate, &s.PeakOutRate)
	if err != nil {
		return nil, fmt.Errorf("get vm network summary: %w", err)
	}
	return &s, nil
}

func (r *pgMetricsRepository) GetNodeNetworkSummary(ctx context.Context, nodeID uuid.UUID, start, end time.Time) (*model.NetworkSummary, error) {
	var s model.NetworkSummary
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(net_in * 60), 0), COALESCE(SUM(net_out * 60), 0),
		        COALESCE(AVG(net_in), 0)::bigint, COALESCE(AVG(net_out), 0)::bigint,
		        COALESCE(MAX(net_in), 0), COALESCE(MAX(net_out), 0)
		 FROM metrics_records
		 WHERE node_id = $1 AND recorded_at BETWEEN $2 AND $3`,
		nodeID, start, end,
	).Scan(&s.TotalIn, &s.TotalOut, &s.AvgInRate, &s.AvgOutRate, &s.PeakInRate, &s.PeakOutRate)
	if err != nil {
		return nil, fmt.Errorf("get node network summary: %w", err)
	}
	return &s, nil
}

func (r *pgMetricsRepository) GetClusterNetworkSummary(ctx context.Context, start, end time.Time) (*model.NetworkSummary, error) {
	var s model.NetworkSummary
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(net_in * 60), 0), COALESCE(SUM(net_out * 60), 0),
		        COALESCE(AVG(net_in), 0)::bigint, COALESCE(AVG(net_out), 0)::bigint,
		        COALESCE(MAX(net_in), 0), COALESCE(MAX(net_out), 0)
		 FROM metrics_records
		 WHERE recorded_at BETWEEN $1 AND $2`,
		start, end,
	).Scan(&s.TotalIn, &s.TotalOut, &s.AvgInRate, &s.AvgOutRate, &s.PeakInRate, &s.PeakOutRate)
	if err != nil {
		return nil, fmt.Errorf("get cluster network summary: %w", err)
	}
	return &s, nil
}

func (r *pgMetricsRepository) CleanupVMMetrics(ctx context.Context, before time.Time) (int64, error) {
	ct, err := r.db.Exec(ctx, "DELETE FROM vm_metrics_history WHERE recorded_at < $1", before)
	if err != nil {
		return 0, fmt.Errorf("cleanup vm metrics: %w", err)
	}
	return ct.RowsAffected(), nil
}
