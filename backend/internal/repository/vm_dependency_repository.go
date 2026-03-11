package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VMDependencyRepository interface {
	Create(ctx context.Context, d *model.VMDependency) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.VMDependency, error)
	List(ctx context.Context) ([]model.VMDependency, error)
	ListByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.VMDependency, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgVMDependencyRepository struct {
	db *pgxpool.Pool
}

func NewVMDependencyRepository(db *pgxpool.Pool) VMDependencyRepository {
	return &pgVMDependencyRepository{db: db}
}

func (r *pgVMDependencyRepository) Create(ctx context.Context, d *model.VMDependency) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO vm_dependencies (source_node_id, source_vmid, target_node_id, target_vmid, dependency_type, description)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		d.SourceNodeID, d.SourceVMID, d.TargetNodeID, d.TargetVMID, d.DependencyType, d.Description,
	).Scan(&d.ID, &d.CreatedAt)
}

func (r *pgVMDependencyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.VMDependency, error) {
	var d model.VMDependency
	err := r.db.QueryRow(ctx,
		`SELECT id, source_node_id, source_vmid, target_node_id, target_vmid, dependency_type, description, created_at
		 FROM vm_dependencies WHERE id = $1`, id,
	).Scan(&d.ID, &d.SourceNodeID, &d.SourceVMID, &d.TargetNodeID, &d.TargetVMID, &d.DependencyType, &d.Description, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get vm dependency: %w", err)
	}
	return &d, nil
}

func (r *pgVMDependencyRepository) List(ctx context.Context) ([]model.VMDependency, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, source_node_id, source_vmid, target_node_id, target_vmid, dependency_type, description, created_at
		 FROM vm_dependencies ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list vm dependencies: %w", err)
	}
	defer rows.Close()

	var deps []model.VMDependency
	for rows.Next() {
		var d model.VMDependency
		if err := rows.Scan(&d.ID, &d.SourceNodeID, &d.SourceVMID, &d.TargetNodeID, &d.TargetVMID, &d.DependencyType, &d.Description, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan vm dependency: %w", err)
		}
		deps = append(deps, d)
	}
	return deps, rows.Err()
}

func (r *pgVMDependencyRepository) ListByVM(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.VMDependency, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, source_node_id, source_vmid, target_node_id, target_vmid, dependency_type, description, created_at
		 FROM vm_dependencies
		 WHERE (source_node_id = $1 AND source_vmid = $2) OR (target_node_id = $1 AND target_vmid = $2)
		 ORDER BY created_at DESC`, nodeID, vmid)
	if err != nil {
		return nil, fmt.Errorf("list vm dependencies by VM: %w", err)
	}
	defer rows.Close()

	var deps []model.VMDependency
	for rows.Next() {
		var d model.VMDependency
		if err := rows.Scan(&d.ID, &d.SourceNodeID, &d.SourceVMID, &d.TargetNodeID, &d.TargetVMID, &d.DependencyType, &d.Description, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan vm dependency: %w", err)
		}
		deps = append(deps, d)
	}
	return deps, rows.Err()
}

func (r *pgVMDependencyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM vm_dependencies WHERE id = $1`, id)
	return err
}
