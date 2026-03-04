package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentConfigRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	List(ctx context.Context) (map[string]string, error)
}

type pgAgentConfigRepository struct {
	db *pgxpool.Pool
}

func NewAgentConfigRepository(db *pgxpool.Pool) AgentConfigRepository {
	return &pgAgentConfigRepository{db: db}
}

func (r *pgAgentConfigRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRow(ctx, "SELECT value FROM agent_config WHERE key = $1", key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("get agent config %s: %w", key, err)
	}
	return value, nil
}

func (r *pgAgentConfigRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO agent_config (key, value, updated_at) VALUES ($1, $2, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("set agent config %s: %w", key, err)
	}
	return nil
}

func (r *pgAgentConfigRepository) List(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.Query(ctx, "SELECT key, value FROM agent_config ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("list agent config: %w", err)
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan agent config: %w", err)
		}
		config[key] = value
	}
	return config, rows.Err()
}
