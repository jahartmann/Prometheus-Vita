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

type TelegramConversationRepository interface {
	Create(ctx context.Context, tc *model.TelegramConversation) error
	GetByChatID(ctx context.Context, chatID int64) (*model.TelegramConversation, error)
	UpdateConversationID(ctx context.Context, id uuid.UUID, convID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgTelegramConversationRepository struct {
	db *pgxpool.Pool
}

func NewTelegramConversationRepository(db *pgxpool.Pool) TelegramConversationRepository {
	return &pgTelegramConversationRepository{db: db}
}

func (r *pgTelegramConversationRepository) Create(ctx context.Context, tc *model.TelegramConversation) error {
	tc.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO telegram_conversations (id, telegram_chat_id, conversation_id, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING created_at, updated_at`,
		tc.ID, tc.TelegramChatID, tc.ConversationID,
	).Scan(&tc.CreatedAt, &tc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create telegram conversation: %w", err)
	}
	return nil
}

func (r *pgTelegramConversationRepository) GetByChatID(ctx context.Context, chatID int64) (*model.TelegramConversation, error) {
	var tc model.TelegramConversation
	err := r.db.QueryRow(ctx,
		`SELECT id, telegram_chat_id, conversation_id, created_at, updated_at
		 FROM telegram_conversations WHERE telegram_chat_id = $1
		 ORDER BY updated_at DESC LIMIT 1`, chatID,
	).Scan(&tc.ID, &tc.TelegramChatID, &tc.ConversationID, &tc.CreatedAt, &tc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get telegram conversation by chat id: %w", err)
	}
	return &tc, nil
}

func (r *pgTelegramConversationRepository) UpdateConversationID(ctx context.Context, id uuid.UUID, convID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE telegram_conversations SET conversation_id=$1, updated_at=NOW() WHERE id=$2`,
		convID, id)
	if err != nil {
		return fmt.Errorf("update telegram conversation: %w", err)
	}
	return nil
}

func (r *pgTelegramConversationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM telegram_conversations WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete telegram conversation: %w", err)
	}
	return nil
}
