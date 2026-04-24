package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserInvitationRepository interface {
	Create(ctx context.Context, invitation *model.UserInvitation) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.UserInvitation, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.UserInvitation, error)
	List(ctx context.Context) ([]model.UserInvitation, error)
	MarkAccepted(ctx context.Context, id uuid.UUID, acceptedAt time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgUserInvitationRepository struct {
	db *pgxpool.Pool
}

func NewUserInvitationRepository(db *pgxpool.Pool) UserInvitationRepository {
	return &pgUserInvitationRepository{db: db}
}

func (r *pgUserInvitationRepository) Create(ctx context.Context, invitation *model.UserInvitation) error {
	invitation.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_invitations
		 (id, username, email, role, token_hash, token_prefix, expires_at, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		invitation.ID,
		invitation.Username,
		invitation.Email,
		invitation.Role,
		invitation.TokenHash,
		invitation.TokenPrefix,
		invitation.ExpiresAt,
		invitation.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create user invitation: %w", err)
	}
	return nil
}

func (r *pgUserInvitationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.UserInvitation, error) {
	return r.getOne(ctx, `WHERE id = $1`, id)
}

func (r *pgUserInvitationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.UserInvitation, error) {
	return r.getOne(ctx, `WHERE token_hash = $1`, tokenHash)
}

func (r *pgUserInvitationRepository) getOne(ctx context.Context, where string, args ...any) (*model.UserInvitation, error) {
	var invitation model.UserInvitation
	query := `SELECT id, username, email, role, token_hash, token_prefix, expires_at, accepted_at, created_by, created_at
		FROM user_invitations ` + where
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&invitation.ID,
		&invitation.Username,
		&invitation.Email,
		&invitation.Role,
		&invitation.TokenHash,
		&invitation.TokenPrefix,
		&invitation.ExpiresAt,
		&invitation.AcceptedAt,
		&invitation.CreatedBy,
		&invitation.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user invitation: %w", err)
	}
	return &invitation, nil
}

func (r *pgUserInvitationRepository) List(ctx context.Context) ([]model.UserInvitation, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, username, email, role, token_hash, token_prefix, expires_at, accepted_at, created_by, created_at
		 FROM user_invitations
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list user invitations: %w", err)
	}
	defer rows.Close()

	var invitations []model.UserInvitation
	for rows.Next() {
		var invitation model.UserInvitation
		if err := rows.Scan(
			&invitation.ID,
			&invitation.Username,
			&invitation.Email,
			&invitation.Role,
			&invitation.TokenHash,
			&invitation.TokenPrefix,
			&invitation.ExpiresAt,
			&invitation.AcceptedAt,
			&invitation.CreatedBy,
			&invitation.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user invitation: %w", err)
		}
		invitations = append(invitations, invitation)
	}
	return invitations, rows.Err()
}

func (r *pgUserInvitationRepository) MarkAccepted(ctx context.Context, id uuid.UUID, acceptedAt time.Time) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE user_invitations SET accepted_at=$1 WHERE id=$2 AND accepted_at IS NULL`,
		acceptedAt,
		id,
	)
	if err != nil {
		return fmt.Errorf("accept user invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgUserInvitationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_invitations WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete user invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
