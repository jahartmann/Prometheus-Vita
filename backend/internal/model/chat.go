package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ChatMessageRole string

const (
	RoleUser      ChatMessageRole = "user"
	RoleAssistant ChatMessageRole = "assistant"
	RoleSystem    ChatMessageRole = "system"
	RoleTool      ChatMessageRole = "tool"
)

type ChatConversation struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChatMessage struct {
	ID             uuid.UUID       `json:"id"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	Role           ChatMessageRole `json:"role"`
	Content        string          `json:"content"`
	ToolCalls      json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID     string          `json:"tool_call_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type AgentToolCall struct {
	ID         uuid.UUID       `json:"id"`
	MessageID  uuid.UUID       `json:"message_id"`
	ToolName   string          `json:"tool_name"`
	Arguments  json.RawMessage `json:"arguments"`
	Result     json.RawMessage `json:"result,omitempty"`
	Status     string          `json:"status"`
	DurationMs int             `json:"duration_ms"`
	CreatedAt  time.Time       `json:"created_at"`
}

// DTOs

type ChatRequest struct {
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	Message        string     `json:"message"`
	Model          string     `json:"model,omitempty"`
}

type ChatResponse struct {
	ConversationID uuid.UUID       `json:"conversation_id"`
	Message        ChatMessage     `json:"message"`
	ToolCalls      []AgentToolCall `json:"tool_calls,omitempty"`
}
