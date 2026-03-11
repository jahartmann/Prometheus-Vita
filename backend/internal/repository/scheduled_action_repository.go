package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduledActionRepository interface {
	Create(ctx context.Context, a *model.ScheduledAction) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.ScheduledAction, error)
	List(ctx context.Context) ([]model.ScheduledAction, error)
	ListByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.ScheduledAction, error)
	ListActive(ctx context.Context) ([]model.ScheduledAction, error)
	Update(ctx context.Context, a *model.ScheduledAction) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgScheduledActionRepository struct {
	db *pgxpool.Pool
}

func NewScheduledActionRepository(db *pgxpool.Pool) ScheduledActionRepository {
	return &pgScheduledActionRepository{db: db}
}

func (r *pgScheduledActionRepository) Create(ctx context.Context, a *model.ScheduledAction) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO scheduled_actions (node_id, vmid, vm_type, action, schedule_cron, is_active, description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		a.NodeID, a.VMID, a.VMType, a.Action, a.ScheduleCron, a.IsActive, a.Description,
	).Scan(&a.ID, &a.CreatedAt)
}

func (r *pgScheduledActionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ScheduledAction, error) {
	var a model.ScheduledAction
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, vmid, vm_type, action, schedule_cron, is_active, description, created_at
		 FROM scheduled_actions WHERE id = $1`, id,
	).Scan(&a.ID, &a.NodeID, &a.VMID, &a.VMType, &a.Action, &a.ScheduleCron, &a.IsActive, &a.Description, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get scheduled action: %w", err)
	}
	return &a, nil
}

func (r *pgScheduledActionRepository) List(ctx context.Context) ([]model.ScheduledAction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, vmid, vm_type, action, schedule_cron, is_active, description, created_at
		 FROM scheduled_actions ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list scheduled actions: %w", err)
	}
	defer rows.Close()

	var actions []model.ScheduledAction
	for rows.Next() {
		var a model.ScheduledAction
		if err := rows.Scan(&a.ID, &a.NodeID, &a.VMID, &a.VMType, &a.Action, &a.ScheduleCron, &a.IsActive, &a.Description, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan scheduled action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *pgScheduledActionRepository) ListByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.ScheduledAction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, vmid, vm_type, action, schedule_cron, is_active, description, created_at
		 FROM scheduled_actions WHERE node_id = $1 AND vmid = $2 ORDER BY created_at DESC`, nodeID, vmid)
	if err != nil {
		return nil, fmt.Errorf("list scheduled actions by VM: %w", err)
	}
	defer rows.Close()

	var actions []model.ScheduledAction
	for rows.Next() {
		var a model.ScheduledAction
		if err := rows.Scan(&a.ID, &a.NodeID, &a.VMID, &a.VMType, &a.Action, &a.ScheduleCron, &a.IsActive, &a.Description, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan scheduled action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *pgScheduledActionRepository) ListActive(ctx context.Context) ([]model.ScheduledAction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, vmid, vm_type, action, schedule_cron, is_active, description, created_at
		 FROM scheduled_actions WHERE is_active = true ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list active scheduled actions: %w", err)
	}
	defer rows.Close()

	var actions []model.ScheduledAction
	for rows.Next() {
		var a model.ScheduledAction
		if err := rows.Scan(&a.ID, &a.NodeID, &a.VMID, &a.VMType, &a.Action, &a.ScheduleCron, &a.IsActive, &a.Description, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan scheduled action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *pgScheduledActionRepository) Update(ctx context.Context, a *model.ScheduledAction) error {
	_, err := r.db.Exec(ctx,
		`UPDATE scheduled_actions SET action = $2, schedule_cron = $3, is_active = $4, description = $5 WHERE id = $1`,
		a.ID, a.Action, a.ScheduleCron, a.IsActive, a.Description)
	return err
}

func (r *pgScheduledActionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM scheduled_actions WHERE id = $1`, id)
	return err
}
