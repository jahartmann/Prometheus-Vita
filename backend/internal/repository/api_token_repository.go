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

type APITokenRepository interface {
	Create(ctx context.Context, token *model.APIToken) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.APIToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.APIToken, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error)
	Update(ctx context.Context, token *model.APIToken) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

type pgAPITokenRepository struct {
	db *pgxpool.Pool
}

func NewAPITokenRepository(db *pgxpool.Pool) APITokenRepository {
	return &pgAPITokenRepository{db: db}
}

func (r *pgAPITokenRepository) Create(ctx context.Context, token *model.APIToken) error {
	token.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, permissions, is_active, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		token.ID, token.UserID, token.Name, token.TokenHash, token.TokenPrefix,
		token.Permissions, token.IsActive, token.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create api token: %w", err)
	}
	return nil
}

func (r *pgAPITokenRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.APIToken, error) {
	var t model.APIToken
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, name, token_hash, token_prefix, permissions, is_active, last_used_at, expires_at, created_at, updated_at
		 FROM api_tokens WHERE id = $1`, id,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.TokenPrefix, &t.Permissions,
		&t.IsActive, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get api token: %w", err)
	}
	return &t, nil
}

func (r *pgAPITokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.APIToken, error) {
	var t model.APIToken
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, name, token_hash, token_prefix, permissions, is_active, last_used_at, expires_at, created_at, updated_at
		 FROM api_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.TokenPrefix, &t.Permissions,
		&t.IsActive, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get api token by hash: %w", err)
	}
	return &t, nil
}

func (r *pgAPITokenRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, token_hash, token_prefix, permissions, is_active, last_used_at, expires_at, created_at, updated_at
		 FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	defer rows.Close()

	var tokens []model.APIToken
	for rows.Next() {
		var t model.APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.TokenPrefix, &t.Permissions,
			&t.IsActive, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan api token: %w", err)
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (r *pgAPITokenRepository) Update(ctx context.Context, token *model.APIToken) error {
	_, err := r.db.Exec(ctx,
		`UPDATE api_tokens SET name=$1, permissions=$2, is_active=$3, expires_at=$4, updated_at=NOW() WHERE id=$5`,
		token.Name, token.Permissions, token.IsActive, token.ExpiresAt, token.ID,
	)
	if err != nil {
		return fmt.Errorf("update api token: %w", err)
	}
	return nil
}

func (r *pgAPITokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM api_tokens WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete api token: %w", err)
	}
	return nil
}

func (r *pgAPITokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "UPDATE api_tokens SET last_used_at=NOW() WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("update api token last used: %w", err)
	}
	return nil
}
