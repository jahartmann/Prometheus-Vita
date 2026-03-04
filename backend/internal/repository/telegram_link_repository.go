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

type TelegramLinkRepository interface {
	Create(ctx context.Context, link *model.TelegramUserLink) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.TelegramUserLink, error)
	GetByTelegramChatID(ctx context.Context, chatID int64) (*model.TelegramUserLink, error)
	GetByVerificationCode(ctx context.Context, code string) (*model.TelegramUserLink, error)
	Verify(ctx context.Context, id uuid.UUID, chatID int64, username string) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type pgTelegramLinkRepository struct {
	db *pgxpool.Pool
}

func NewTelegramLinkRepository(db *pgxpool.Pool) TelegramLinkRepository {
	return &pgTelegramLinkRepository{db: db}
}

func (r *pgTelegramLinkRepository) Create(ctx context.Context, link *model.TelegramUserLink) error {
	link.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO telegram_user_links (id, user_id, verification_code, is_verified, created_at)
		 VALUES ($1, $2, $3, false, NOW())
		 RETURNING created_at`,
		link.ID, link.UserID, link.VerificationCode,
	).Scan(&link.CreatedAt)
	if err != nil {
		return fmt.Errorf("create telegram link: %w", err)
	}
	return nil
}

func (r *pgTelegramLinkRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.TelegramUserLink, error) {
	var l model.TelegramUserLink
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, telegram_chat_id, telegram_username, verification_code,
		        is_verified, created_at, verified_at
		 FROM telegram_user_links WHERE user_id = $1`, userID,
	).Scan(&l.ID, &l.UserID, &l.TelegramChatID, &l.TelegramUsername, &l.VerificationCode,
		&l.IsVerified, &l.CreatedAt, &l.VerifiedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get telegram link by user: %w", err)
	}
	return &l, nil
}

func (r *pgTelegramLinkRepository) GetByTelegramChatID(ctx context.Context, chatID int64) (*model.TelegramUserLink, error) {
	var l model.TelegramUserLink
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, telegram_chat_id, telegram_username, verification_code,
		        is_verified, created_at, verified_at
		 FROM telegram_user_links WHERE telegram_chat_id = $1 AND is_verified = true`, chatID,
	).Scan(&l.ID, &l.UserID, &l.TelegramChatID, &l.TelegramUsername, &l.VerificationCode,
		&l.IsVerified, &l.CreatedAt, &l.VerifiedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get telegram link by chat id: %w", err)
	}
	return &l, nil
}

func (r *pgTelegramLinkRepository) GetByVerificationCode(ctx context.Context, code string) (*model.TelegramUserLink, error) {
	var l model.TelegramUserLink
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, telegram_chat_id, telegram_username, verification_code,
		        is_verified, created_at, verified_at
		 FROM telegram_user_links WHERE verification_code = $1 AND is_verified = false`, code,
	).Scan(&l.ID, &l.UserID, &l.TelegramChatID, &l.TelegramUsername, &l.VerificationCode,
		&l.IsVerified, &l.CreatedAt, &l.VerifiedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get telegram link by verification code: %w", err)
	}
	return &l, nil
}

func (r *pgTelegramLinkRepository) Verify(ctx context.Context, id uuid.UUID, chatID int64, username string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE telegram_user_links
		 SET telegram_chat_id=$1, telegram_username=$2, is_verified=true, verified_at=NOW()
		 WHERE id=$3`,
		chatID, username, id)
	if err != nil {
		return fmt.Errorf("verify telegram link: %w", err)
	}
	return nil
}

func (r *pgTelegramLinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM telegram_user_links WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete telegram link: %w", err)
	}
	return nil
}

func (r *pgTelegramLinkRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM telegram_user_links WHERE user_id=$1", userID)
	if err != nil {
		return fmt.Errorf("delete telegram link by user: %w", err)
	}
	return nil
}
