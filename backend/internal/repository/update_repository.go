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

type UpdateRepository interface {
	Create(ctx context.Context, check *model.UpdateCheck) error
	GetLatestByNode(ctx context.Context, nodeID uuid.UUID) (*model.UpdateCheck, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.UpdateCheck, error)
	ListAll(ctx context.Context, limit int) ([]model.UpdateCheck, error)
	Update(ctx context.Context, check *model.UpdateCheck) error
}

type pgUpdateRepository struct {
	db *pgxpool.Pool
}

func NewUpdateRepository(db *pgxpool.Pool) UpdateRepository {
	return &pgUpdateRepository{db: db}
}

func (r *pgUpdateRepository) Create(ctx context.Context, check *model.UpdateCheck) error {
	check.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO update_checks (id, node_id, status, total_updates, security_updates, packages, error_message, checked_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		check.ID, check.NodeID, check.Status, check.TotalUpdates, check.SecurityUpdates,
		check.Packages, check.ErrorMessage, check.CheckedAt,
	)
	if err != nil {
		return fmt.Errorf("create update check: %w", err)
	}
	return nil
}

func (r *pgUpdateRepository) GetLatestByNode(ctx context.Context, nodeID uuid.UUID) (*model.UpdateCheck, error) {
	var c model.UpdateCheck
	err := r.db.QueryRow(ctx,
		`SELECT id, node_id, status, total_updates, security_updates, packages, error_message, checked_at, created_at
		 FROM update_checks WHERE node_id = $1 ORDER BY checked_at DESC LIMIT 1`, nodeID,
	).Scan(&c.ID, &c.NodeID, &c.Status, &c.TotalUpdates, &c.SecurityUpdates,
		&c.Packages, &c.ErrorMessage, &c.CheckedAt, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get latest update check: %w", err)
	}
	return &c, nil
}

func (r *pgUpdateRepository) ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.UpdateCheck, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, status, total_updates, security_updates, packages, error_message, checked_at, created_at
		 FROM update_checks WHERE node_id = $1 ORDER BY checked_at DESC LIMIT $2`, nodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list update checks: %w", err)
	}
	defer rows.Close()

	var checks []model.UpdateCheck
	for rows.Next() {
		var c model.UpdateCheck
		if err := rows.Scan(&c.ID, &c.NodeID, &c.Status, &c.TotalUpdates, &c.SecurityUpdates,
			&c.Packages, &c.ErrorMessage, &c.CheckedAt, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan update check: %w", err)
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (r *pgUpdateRepository) ListAll(ctx context.Context, limit int) ([]model.UpdateCheck, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, node_id, status, total_updates, security_updates, packages, error_message, checked_at, created_at
		 FROM update_checks ORDER BY checked_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list all update checks: %w", err)
	}
	defer rows.Close()

	var checks []model.UpdateCheck
	for rows.Next() {
		var c model.UpdateCheck
		if err := rows.Scan(&c.ID, &c.NodeID, &c.Status, &c.TotalUpdates, &c.SecurityUpdates,
			&c.Packages, &c.ErrorMessage, &c.CheckedAt, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan update check: %w", err)
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (r *pgUpdateRepository) Update(ctx context.Context, check *model.UpdateCheck) error {
	_, err := r.db.Exec(ctx,
		`UPDATE update_checks SET status=$1, total_updates=$2, security_updates=$3, packages=$4, error_message=$5, checked_at=$6
		 WHERE id=$7`,
		check.Status, check.TotalUpdates, check.SecurityUpdates, check.Packages, check.ErrorMessage, check.CheckedAt, check.ID,
	)
	if err != nil {
		return fmt.Errorf("update update check: %w", err)
	}
	return nil
}
