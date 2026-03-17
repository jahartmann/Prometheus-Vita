package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LogSourceRepository interface {
	Upsert(ctx context.Context, source *model.LogSource) error
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.LogSource, error)
	UpdateEnabled(ctx context.Context, nodeID uuid.UUID, path string, enabled bool) error
}

type pgLogSourceRepository struct {
	db *pgxpool.Pool
}

func NewLogSourceRepository(db *pgxpool.Pool) LogSourceRepository {
	return &pgLogSourceRepository{db: db}
}

func (r *pgLogSourceRepository) Upsert(ctx context.Context, source *model.LogSource) error {
	if source.ID == uuid.Nil {
		source.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO log_sources (id, node_id, path, enabled, is_builtin, parser_type, discovered_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (node_id, path) DO UPDATE SET enabled=EXCLUDED.enabled, parser_type=EXCLUDED.parser_type`,
		source.ID, source.NodeID, source.Path, source.Enabled, source.IsBuiltin, source.ParserType,
	)
	if err != nil {
		return fmt.Errorf("upsert log source: %w", err)
	}
	return nil
}

func (r *pgLogSourceRepository) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.LogSource, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, path, enabled, is_builtin, parser_type, discovered_at
		 FROM log_sources WHERE node_id = $1 ORDER BY path ASC`,
		nodeID)
	if err != nil {
		return nil, fmt.Errorf("list log sources: %w", err)
	}
	defer rows.Close()

	var sources []model.LogSource
	for rows.Next() {
		var s model.LogSource
		if err := rows.Scan(&s.ID, &s.NodeID, &s.Path, &s.Enabled, &s.IsBuiltin, &s.ParserType, &s.DiscoveredAt); err != nil {
			return nil, fmt.Errorf("scan log source: %w", err)
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func (r *pgLogSourceRepository) UpdateEnabled(ctx context.Context, nodeID uuid.UUID, path string, enabled bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE log_sources SET enabled=$3 WHERE node_id=$1 AND path=$2`,
		nodeID, path, enabled,
	)
	if err != nil {
		return fmt.Errorf("update log source enabled: %w", err)
	}
	return nil
}
