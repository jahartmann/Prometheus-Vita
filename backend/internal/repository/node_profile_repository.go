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

type NodeProfileRepository interface {
	Create(ctx context.Context, profile *model.NodeProfile) error
	GetLatest(ctx context.Context, nodeID uuid.UUID) (*model.NodeProfile, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.NodeProfile, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgNodeProfileRepository struct {
	db *pgxpool.Pool
}

func NewNodeProfileRepository(db *pgxpool.Pool) NodeProfileRepository {
	return &pgNodeProfileRepository{db: db}
}

func (r *pgNodeProfileRepository) Create(ctx context.Context, profile *model.NodeProfile) error {
	profile.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO node_profiles (id, node_id, collected_at, cpu_model, cpu_cores, cpu_threads,
		        memory_total_bytes, memory_modules, disks, network_interfaces,
		        pve_version, kernel_version, installed_packages, storage_layout, custom_data)
		 VALUES ($1, $2, NOW(), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		profile.ID, profile.NodeID,
		profile.CPUModel, profile.CPUCores, profile.CPUThreads,
		profile.MemoryTotalBytes, profile.MemoryModules,
		profile.Disks, profile.NetworkInterfaces,
		profile.PVEVersion, profile.KernelVersion,
		profile.InstalledPackages, profile.StorageLayout, profile.CustomData,
	)
	if err != nil {
		return fmt.Errorf("create node profile: %w", err)
	}
	return nil
}

func (r *pgNodeProfileRepository) GetLatest(ctx context.Context, nodeID uuid.UUID) (*model.NodeProfile, error) {
	var p model.NodeProfile
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, collected_at, cpu_model, cpu_cores, cpu_threads,
		        memory_total_bytes, memory_modules, disks, network_interfaces,
		        pve_version, kernel_version, installed_packages, storage_layout, custom_data
		 FROM node_profiles WHERE node_id = $1 ORDER BY collected_at DESC LIMIT 1`, nodeID,
	).Scan(&p.ID, &p.NodeID, &p.CollectedAt,
		&p.CPUModel, &p.CPUCores, &p.CPUThreads,
		&p.MemoryTotalBytes, &p.MemoryModules,
		&p.Disks, &p.NetworkInterfaces,
		&p.PVEVersion, &p.KernelVersion,
		&p.InstalledPackages, &p.StorageLayout, &p.CustomData)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get latest node profile: %w", err)
	}
	return &p, nil
}

func (r *pgNodeProfileRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.NodeProfile, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, collected_at, cpu_model, cpu_cores, cpu_threads,
		        memory_total_bytes, memory_modules, disks, network_interfaces,
		        pve_version, kernel_version, installed_packages, storage_layout, custom_data
		 FROM node_profiles WHERE node_id = $1 ORDER BY collected_at DESC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list node profiles: %w", err)
	}
	defer rows.Close()

	var profiles []model.NodeProfile
	for rows.Next() {
		var p model.NodeProfile
		if err := rows.Scan(&p.ID, &p.NodeID, &p.CollectedAt,
			&p.CPUModel, &p.CPUCores, &p.CPUThreads,
			&p.MemoryTotalBytes, &p.MemoryModules,
			&p.Disks, &p.NetworkInterfaces,
			&p.PVEVersion, &p.KernelVersion,
			&p.InstalledPackages, &p.StorageLayout, &p.CustomData); err != nil {
			return nil, fmt.Errorf("scan node profile: %w", err)
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (r *pgNodeProfileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM node_profiles WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete node profile: %w", err)
	}
	return nil
}
