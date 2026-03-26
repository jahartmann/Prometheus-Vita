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

type NodeRepository interface {
	Create(ctx context.Context, node *model.Node) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Node, error)
	List(ctx context.Context) ([]model.Node, error)
	Update(ctx context.Context, node *model.Node) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, isOnline bool) error
	UpdateEnvironment(ctx context.Context, nodeID uuid.UUID, envID *uuid.UUID) error
	UpdateSSHHostKey(ctx context.Context, nodeID uuid.UUID, hostKey string) error
	ListByEnvironment(ctx context.Context, envID uuid.UUID) ([]model.Node, error)
}

type pgNodeRepository struct {
	db *pgxpool.Pool
}

func NewNodeRepository(db *pgxpool.Pool) NodeRepository {
	return &pgNodeRepository{db: db}
}

func (r *pgNodeRepository) Create(ctx context.Context, node *model.Node) error {
	node.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO nodes (id, name, type, hostname, port, api_token_id, api_token_secret,
		        ssh_port, ssh_user, ssh_private_key, ssh_host_key, is_online, metadata, environment_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())`,
		node.ID, node.Name, node.Type, node.Hostname, node.Port,
		node.APITokenID, node.APITokenSecret,
		node.SSHPort, node.SSHUser, node.SSHPrivateKey, node.SSHHostKey,
		node.IsOnline, node.Metadata, node.EnvironmentID,
	)
	if err != nil {
		return fmt.Errorf("create node: %w", err)
	}
	return nil
}

func (r *pgNodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Node, error) {
	var n model.Node
	err := r.db.QueryRow(ctx,
		`SELECT id, name, type, hostname, port, api_token_id, api_token_secret,
		        ssh_port, COALESCE(ssh_user, ''), COALESCE(ssh_private_key, ''),
		        COALESCE(ssh_host_key, ''),
		        is_online, last_seen, metadata, environment_id, created_at, updated_at
		 FROM nodes WHERE id = $1`, id,
	).Scan(&n.ID, &n.Name, &n.Type, &n.Hostname, &n.Port,
		&n.APITokenID, &n.APITokenSecret,
		&n.SSHPort, &n.SSHUser, &n.SSHPrivateKey, &n.SSHHostKey,
		&n.IsOnline, &n.LastSeen,
		&n.Metadata, &n.EnvironmentID, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get node by id: %w", err)
	}
	return &n, nil
}

func (r *pgNodeRepository) List(ctx context.Context) ([]model.Node, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, type, hostname, port, api_token_id, api_token_secret,
		        ssh_port, COALESCE(ssh_user, ''), COALESCE(ssh_private_key, ''),
		        COALESCE(ssh_host_key, ''),
		        is_online, last_seen, metadata, environment_id, created_at, updated_at
		 FROM nodes ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []model.Node
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.Hostname, &n.Port,
			&n.APITokenID, &n.APITokenSecret,
			&n.SSHPort, &n.SSHUser, &n.SSHPrivateKey, &n.SSHHostKey,
			&n.IsOnline, &n.LastSeen,
			&n.Metadata, &n.EnvironmentID, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (r *pgNodeRepository) Update(ctx context.Context, node *model.Node) error {
	_, err := r.db.Exec(ctx,
		`UPDATE nodes SET name=$1, hostname=$2, port=$3, api_token_id=$4,
		        api_token_secret=$5, ssh_port=$6, ssh_user=$7, ssh_private_key=$8,
		        ssh_host_key=$9, metadata=$10, environment_id=$11, updated_at=NOW()
		 WHERE id=$12`,
		node.Name, node.Hostname, node.Port,
		node.APITokenID, node.APITokenSecret,
		node.SSHPort, node.SSHUser, node.SSHPrivateKey, node.SSHHostKey,
		node.Metadata, node.EnvironmentID, node.ID,
	)
	if err != nil {
		return fmt.Errorf("update node: %w", err)
	}
	return nil
}

func (r *pgNodeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM nodes WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete node: %w", err)
	}
	return nil
}

func (r *pgNodeRepository) UpdateStatus(ctx context.Context, id uuid.UUID, isOnline bool) error {
	_, err := r.db.Exec(ctx,
		"UPDATE nodes SET is_online=$1, last_seen=NOW(), updated_at=NOW() WHERE id=$2",
		isOnline, id,
	)
	if err != nil {
		return fmt.Errorf("update node status: %w", err)
	}
	return nil
}

func (r *pgNodeRepository) UpdateEnvironment(ctx context.Context, nodeID uuid.UUID, envID *uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE nodes SET environment_id=$1, updated_at=NOW() WHERE id=$2",
		envID, nodeID,
	)
	if err != nil {
		return fmt.Errorf("update node environment: %w", err)
	}
	return nil
}

func (r *pgNodeRepository) UpdateSSHHostKey(ctx context.Context, nodeID uuid.UUID, hostKey string) error {
	_, err := r.db.Exec(ctx,
		"UPDATE nodes SET ssh_host_key=$1, updated_at=NOW() WHERE id=$2",
		hostKey, nodeID,
	)
	if err != nil {
		return fmt.Errorf("update node ssh host key: %w", err)
	}
	return nil
}

func (r *pgNodeRepository) ListByEnvironment(ctx context.Context, envID uuid.UUID) ([]model.Node, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, type, hostname, port, api_token_id, api_token_secret,
		        ssh_port, COALESCE(ssh_user, ''), COALESCE(ssh_private_key, ''),
		        COALESCE(ssh_host_key, ''),
		        is_online, last_seen, metadata, environment_id, created_at, updated_at
		 FROM nodes WHERE environment_id = $1 ORDER BY name ASC`, envID)
	if err != nil {
		return nil, fmt.Errorf("list nodes by environment: %w", err)
	}
	defer rows.Close()

	var nodes []model.Node
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.Hostname, &n.Port,
			&n.APITokenID, &n.APITokenSecret,
			&n.SSHPort, &n.SSHUser, &n.SSHPrivateKey, &n.SSHHostKey,
			&n.IsOnline, &n.LastSeen,
			&n.Metadata, &n.EnvironmentID, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}
