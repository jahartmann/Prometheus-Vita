package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	Revoked   bool
}

var ErrAlreadyRevoked = fmt.Errorf("token already revoked")

type TokenRepository interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.UserSession, error)
	RevokeByHash(ctx context.Context, tokenHash string) error
	RevokeByIDForUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

type pgTokenRepository struct {
	db *pgxpool.Pool
}

func NewTokenRepository(db *pgxpool.Pool) TokenRepository {
	return &pgTokenRepository{db: db}
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (r *pgTokenRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *pgTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	var t RefreshToken
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at, revoked
		 FROM refresh_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt, &t.Revoked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &t, nil
}

func (r *pgTokenRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.UserSession, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, expires_at, created_at, revoked,
		        (revoked = false AND expires_at > NOW()) AS is_active
		 FROM refresh_tokens
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []model.UserSession
	for rows.Next() {
		var s model.UserSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt, &s.Revoked, &s.IsActive); err != nil {
			return nil, fmt.Errorf("scan user session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *pgTokenRepository) RevokeByHash(ctx context.Context, tokenHash string) error {
	tag, err := r.db.Exec(ctx,
		"UPDATE refresh_tokens SET revoked=true WHERE token_hash=$1 AND revoked=false", tokenHash,
	)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyRevoked
	}
	return nil
}

func (r *pgTokenRepository) RevokeByIDForUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		"UPDATE refresh_tokens SET revoked=true WHERE id=$1 AND user_id=$2 AND revoked=false",
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke user session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE refresh_tokens SET revoked=true WHERE user_id=$1 AND revoked=false", userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all tokens for user: %w", err)
	}
	return nil
}

func (r *pgTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM refresh_tokens WHERE expires_at < NOW() OR revoked = true",
	)
	if err != nil {
		return fmt.Errorf("delete expired tokens: %w", err)
	}
	return nil
}
