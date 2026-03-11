package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SnapshotPolicyRepository interface {
	Create(ctx context.Context, p *model.SnapshotPolicy) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.SnapshotPolicy, error)
	List(ctx context.Context) ([]model.SnapshotPolicy, error)
	ListByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.SnapshotPolicy, error)
	ListActive(ctx context.Context) ([]model.SnapshotPolicy, error)
	Update(ctx context.Context, p *model.SnapshotPolicy) error
	UpdateLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgSnapshotPolicyRepository struct {
	db *pgxpool.Pool
}

func NewSnapshotPolicyRepository(db *pgxpool.Pool) SnapshotPolicyRepository {
	return &pgSnapshotPolicyRepository{db: db}
}

func (r *pgSnapshotPolicyRepository) Create(ctx context.Context, p *model.SnapshotPolicy) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO snapshot_policies (node_id, vmid, vm_type, name, keep_daily, keep_weekly, keep_monthly, schedule_cron, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at`,
		p.NodeID, p.VMID, p.VMType, p.Name, p.KeepDaily, p.KeepWeekly, p.KeepMonthly, p.ScheduleCron, p.IsActive,
	).Scan(&p.ID, &p.CreatedAt)
}

func (r *pgSnapshotPolicyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.SnapshotPolicy, error) {
	var p model.SnapshotPolicy
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, vmid, vm_type, name, keep_daily, keep_weekly, keep_monthly, schedule_cron, is_active, last_run, created_at
		 FROM snapshot_policies WHERE id = $1`, id,
	).Scan(&p.ID, &p.NodeID, &p.VMID, &p.VMType, &p.Name, &p.KeepDaily, &p.KeepWeekly, &p.KeepMonthly, &p.ScheduleCron, &p.IsActive, &p.LastRun, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get snapshot policy: %w", err)
	}
	return &p, nil
}

func (r *pgSnapshotPolicyRepository) List(ctx context.Context) ([]model.SnapshotPolicy, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, vmid, vm_type, name, keep_daily, keep_weekly, keep_monthly, schedule_cron, is_active, last_run, created_at
		 FROM snapshot_policies ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list snapshot policies: %w", err)
	}
	defer rows.Close()

	var policies []model.SnapshotPolicy
	for rows.Next() {
		var p model.SnapshotPolicy
		if err := rows.Scan(&p.ID, &p.NodeID, &p.VMID, &p.VMType, &p.Name, &p.KeepDaily, &p.KeepWeekly, &p.KeepMonthly, &p.ScheduleCron, &p.IsActive, &p.LastRun, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot policy: %w", err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (r *pgSnapshotPolicyRepository) ListByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.SnapshotPolicy, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, vmid, vm_type, name, keep_daily, keep_weekly, keep_monthly, schedule_cron, is_active, last_run, created_at
		 FROM snapshot_policies WHERE node_id = $1 AND vmid = $2 ORDER BY created_at DESC`, nodeID, vmid)
	if err != nil {
		return nil, fmt.Errorf("list snapshot policies by VM: %w", err)
	}
	defer rows.Close()

	var policies []model.SnapshotPolicy
	for rows.Next() {
		var p model.SnapshotPolicy
		if err := rows.Scan(&p.ID, &p.NodeID, &p.VMID, &p.VMType, &p.Name, &p.KeepDaily, &p.KeepWeekly, &p.KeepMonthly, &p.ScheduleCron, &p.IsActive, &p.LastRun, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot policy: %w", err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (r *pgSnapshotPolicyRepository) ListActive(ctx context.Context) ([]model.SnapshotPolicy, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, vmid, vm_type, name, keep_daily, keep_weekly, keep_monthly, schedule_cron, is_active, last_run, created_at
		 FROM snapshot_policies WHERE is_active = true ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list active snapshot policies: %w", err)
	}
	defer rows.Close()

	var policies []model.SnapshotPolicy
	for rows.Next() {
		var p model.SnapshotPolicy
		if err := rows.Scan(&p.ID, &p.NodeID, &p.VMID, &p.VMType, &p.Name, &p.KeepDaily, &p.KeepWeekly, &p.KeepMonthly, &p.ScheduleCron, &p.IsActive, &p.LastRun, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot policy: %w", err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (r *pgSnapshotPolicyRepository) Update(ctx context.Context, p *model.SnapshotPolicy) error {
	_, err := r.db.Exec(ctx,
		`UPDATE snapshot_policies SET name = $2, keep_daily = $3, keep_weekly = $4, keep_monthly = $5, schedule_cron = $6, is_active = $7 WHERE id = $1`,
		p.ID, p.Name, p.KeepDaily, p.KeepWeekly, p.KeepMonthly, p.ScheduleCron, p.IsActive)
	return err
}

func (r *pgSnapshotPolicyRepository) UpdateLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE snapshot_policies SET last_run = $2 WHERE id = $1`, id, lastRun)
	return err
}

func (r *pgSnapshotPolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM snapshot_policies WHERE id = $1`, id)
	return err
}
