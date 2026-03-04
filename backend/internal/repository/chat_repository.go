package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChatConversationRepository manages chat conversations.
type ChatConversationRepository interface {
	Create(ctx context.Context, conv *model.ChatConversation) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.ChatConversation, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.ChatConversation, error)
	UpdateTitle(ctx context.Context, id uuid.UUID, title string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ChatMessageRepository manages chat messages.
type ChatMessageRepository interface {
	Create(ctx context.Context, msg *model.ChatMessage) error
	ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]model.ChatMessage, error)
	CountByConversation(ctx context.Context, conversationID uuid.UUID) (int, error)
}

// ToolCallRepository manages agent tool calls.
type ToolCallRepository interface {
	Create(ctx context.Context, tc *model.AgentToolCall) error
	UpdateResult(ctx context.Context, id uuid.UUID, result json.RawMessage, status string, durationMs int) error
	ListByMessage(ctx context.Context, messageID uuid.UUID) ([]model.AgentToolCall, error)
}

// --- ChatConversationRepository implementation ---

type pgChatConversationRepository struct {
	db *pgxpool.Pool
}

func NewChatConversationRepository(db *pgxpool.Pool) ChatConversationRepository {
	return &pgChatConversationRepository{db: db}
}

func (r *pgChatConversationRepository) Create(ctx context.Context, conv *model.ChatConversation) error {
	conv.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO chat_conversations (id, user_id, title, model, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 RETURNING created_at, updated_at`,
		conv.ID, conv.UserID, conv.Title, conv.Model,
	).Scan(&conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create chat conversation: %w", err)
	}
	return nil
}

func (r *pgChatConversationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ChatConversation, error) {
	var conv model.ChatConversation
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, title, model, created_at, updated_at
		 FROM chat_conversations WHERE id = $1`, id,
	).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Model, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get chat conversation: %w", err)
	}
	return &conv, nil
}

func (r *pgChatConversationRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.ChatConversation, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, title, model, created_at, updated_at
		 FROM chat_conversations WHERE user_id = $1
		 ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list chat conversations: %w", err)
	}
	defer rows.Close()

	var convs []model.ChatConversation
	for rows.Next() {
		var conv model.ChatConversation
		if err := rows.Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Model, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan chat conversation: %w", err)
		}
		convs = append(convs, conv)
	}
	return convs, rows.Err()
}

func (r *pgChatConversationRepository) UpdateTitle(ctx context.Context, id uuid.UUID, title string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE chat_conversations SET title = $1, updated_at = NOW() WHERE id = $2`,
		title, id)
	if err != nil {
		return fmt.Errorf("update chat conversation title: %w", err)
	}
	return nil
}

func (r *pgChatConversationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM chat_conversations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete chat conversation: %w", err)
	}
	return nil
}

// --- ChatMessageRepository implementation ---

type pgChatMessageRepository struct {
	db *pgxpool.Pool
}

func NewChatMessageRepository(db *pgxpool.Pool) ChatMessageRepository {
	return &pgChatMessageRepository{db: db}
}

func (r *pgChatMessageRepository) Create(ctx context.Context, msg *model.ChatMessage) error {
	msg.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO chat_messages (id, conversation_id, role, content, tool_calls, tool_call_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 RETURNING created_at`,
		msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.ToolCalls, msg.ToolCallID,
	).Scan(&msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("create chat message: %w", err)
	}
	return nil
}

func (r *pgChatMessageRepository) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]model.ChatMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, conversation_id, role, content, tool_calls, tool_call_id, created_at
		 FROM chat_messages WHERE conversation_id = $1
		 ORDER BY created_at ASC`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}
	defer rows.Close()

	var msgs []model.ChatMessage
	for rows.Next() {
		var msg model.ChatMessage
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content,
			&msg.ToolCalls, &msg.ToolCallID, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

func (r *pgChatMessageRepository) CountByConversation(ctx context.Context, conversationID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM chat_messages WHERE conversation_id = $1`, conversationID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count chat messages: %w", err)
	}
	return count, nil
}

// --- ToolCallRepository implementation ---

type pgToolCallRepository struct {
	db *pgxpool.Pool
}

func NewToolCallRepository(db *pgxpool.Pool) ToolCallRepository {
	return &pgToolCallRepository{db: db}
}

func (r *pgToolCallRepository) Create(ctx context.Context, tc *model.AgentToolCall) error {
	tc.ID = uuid.New()
	err := r.db.QueryRow(ctx,
		`INSERT INTO agent_tool_calls (id, message_id, tool_name, arguments, result, status, duration_ms, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 RETURNING created_at`,
		tc.ID, tc.MessageID, tc.ToolName, tc.Arguments, tc.Result, tc.Status, tc.DurationMs,
	).Scan(&tc.CreatedAt)
	if err != nil {
		return fmt.Errorf("create tool call: %w", err)
	}
	return nil
}

func (r *pgToolCallRepository) UpdateResult(ctx context.Context, id uuid.UUID, result json.RawMessage, status string, durationMs int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE agent_tool_calls SET result = $1, status = $2, duration_ms = $3 WHERE id = $4`,
		result, status, durationMs, id)
	if err != nil {
		return fmt.Errorf("update tool call result: %w", err)
	}
	return nil
}

func (r *pgToolCallRepository) ListByMessage(ctx context.Context, messageID uuid.UUID) ([]model.AgentToolCall, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, message_id, tool_name, arguments, result, status, duration_ms, created_at
		 FROM agent_tool_calls WHERE message_id = $1
		 ORDER BY created_at ASC`, messageID)
	if err != nil {
		return nil, fmt.Errorf("list tool calls: %w", err)
	}
	defer rows.Close()

	var tcs []model.AgentToolCall
	for rows.Next() {
		var tc model.AgentToolCall
		if err := rows.Scan(&tc.ID, &tc.MessageID, &tc.ToolName, &tc.Arguments,
			&tc.Result, &tc.Status, &tc.DurationMs, &tc.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tool call: %w", err)
		}
		tcs = append(tcs, tc)
	}
	return tcs, rows.Err()
}
