package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NetworkAliasRepository interface {
	Upsert(ctx context.Context, alias *model.NetworkAlias) error
	GetByNode(ctx context.Context, nodeID uuid.UUID) ([]model.NetworkAlias, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgNetworkAliasRepository struct {
	db *pgxpool.Pool
}

func NewNetworkAliasRepository(db *pgxpool.Pool) NetworkAliasRepository {
	return &pgNetworkAliasRepository{db: db}
}

func (r *pgNetworkAliasRepository) Upsert(ctx context.Context, alias *model.NetworkAlias) error {
	alias.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO network_aliases (id, node_id, interface_name, display_name, description, color, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 ON CONFLICT (node_id, interface_name)
		 DO UPDATE SET display_name=$4, description=$5, color=$6, updated_at=NOW()`,
		alias.ID, alias.NodeID, alias.InterfaceName,
		alias.DisplayName, alias.Description, alias.Color,
	)
	if err != nil {
		return fmt.Errorf("upsert network alias: %w", err)
	}
	return nil
}

func (r *pgNetworkAliasRepository) GetByNode(ctx context.Context, nodeID uuid.UUID) ([]model.NetworkAlias, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, interface_name, display_name, description, color, created_at, updated_at
		 FROM network_aliases WHERE node_id = $1 ORDER BY interface_name ASC`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get network aliases by node: %w", err)
	}
	defer rows.Close()

	var aliases []model.NetworkAlias
	for rows.Next() {
		var a model.NetworkAlias
		if err := rows.Scan(&a.ID, &a.NodeID, &a.InterfaceName, &a.DisplayName,
			&a.Description, &a.Color, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan network alias: %w", err)
		}
		aliases = append(aliases, a)
	}
	return aliases, rows.Err()
}

func (r *pgNetworkAliasRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM network_aliases WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete network alias: %w", err)
	}
	return nil
}
