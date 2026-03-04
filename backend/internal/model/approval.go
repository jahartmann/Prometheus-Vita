package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

type AgentPendingApproval struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	MessageID      uuid.UUID       `json:"message_id"`
	ToolName       string          `json:"tool_name"`
	Arguments      json.RawMessage `json:"arguments"`
	Status         ApprovalStatus  `json:"status"`
	ResolvedBy     *uuid.UUID      `json:"resolved_by,omitempty"`
	ResolvedAt     *time.Time      `json:"resolved_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}
