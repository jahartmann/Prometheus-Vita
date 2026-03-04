package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Channel types

type NotificationChannelType string

const (
	ChannelTypeEmail    NotificationChannelType = "email"
	ChannelTypeTelegram NotificationChannelType = "telegram"
	ChannelTypeWebhook  NotificationChannelType = "webhook"
)

func (t NotificationChannelType) IsValid() bool {
	switch t {
	case ChannelTypeEmail, ChannelTypeTelegram, ChannelTypeWebhook:
		return true
	}
	return false
}

// Notification status

type NotificationStatus string

const (
	NotifStatusPending NotificationStatus = "pending"
	NotifStatusSent    NotificationStatus = "sent"
	NotifStatusFailed  NotificationStatus = "failed"
)

// Alert severity

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

func (s AlertSeverity) IsValid() bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	}
	return false
}

// Domain models

type NotificationChannel struct {
	ID        uuid.UUID               `json:"id"`
	Name      string                  `json:"name"`
	Type      NotificationChannelType `json:"type"`
	Config    json.RawMessage         `json:"config"`
	IsActive  bool                    `json:"is_active"`
	CreatedBy *uuid.UUID              `json:"created_by,omitempty"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type NotificationHistoryEntry struct {
	ID           uuid.UUID          `json:"id"`
	ChannelID    *uuid.UUID         `json:"channel_id,omitempty"`
	EventType    string             `json:"event_type"`
	Subject      string             `json:"subject"`
	Body         string             `json:"body"`
	Status       NotificationStatus `json:"status"`
	ErrorMessage string             `json:"error_message,omitempty"`
	Metadata     json.RawMessage    `json:"metadata,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	SentAt       *time.Time         `json:"sent_at,omitempty"`
}

type AlertRule struct {
	ID                 uuid.UUID     `json:"id"`
	Name               string        `json:"name"`
	NodeID             uuid.UUID     `json:"node_id"`
	Metric             string        `json:"metric"`
	Operator           string        `json:"operator"`
	Threshold          float64       `json:"threshold"`
	DurationSeconds    int           `json:"duration_seconds"`
	Severity           AlertSeverity `json:"severity"`
	ChannelIDs         []uuid.UUID   `json:"channel_ids"`
	EscalationPolicyID *uuid.UUID    `json:"escalation_policy_id,omitempty"`
	IsActive           bool          `json:"is_active"`
	LastTriggeredAt    *time.Time    `json:"last_triggered_at,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// Incident status
type IncidentStatus string

const (
	IncidentStatusTriggered    IncidentStatus = "triggered"
	IncidentStatusAcknowledged IncidentStatus = "acknowledged"
	IncidentStatusResolved     IncidentStatus = "resolved"
)

// Escalation models

type EscalationPolicy struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	IsActive    bool             `json:"is_active"`
	Steps       []EscalationStep `json:"steps,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type EscalationStep struct {
	ID           uuid.UUID   `json:"id"`
	PolicyID     uuid.UUID   `json:"policy_id"`
	StepOrder    int         `json:"step_order"`
	DelaySeconds int         `json:"delay_seconds"`
	ChannelIDs   []uuid.UUID `json:"channel_ids"`
	CreatedAt    time.Time   `json:"created_at"`
}

type AlertIncident struct {
	ID              uuid.UUID      `json:"id"`
	AlertRuleID     uuid.UUID      `json:"alert_rule_id"`
	Status          IncidentStatus `json:"status"`
	CurrentStep     int            `json:"current_step"`
	TriggeredAt     time.Time      `json:"triggered_at"`
	AcknowledgedAt  *time.Time     `json:"acknowledged_at,omitempty"`
	AcknowledgedBy  *uuid.UUID     `json:"acknowledged_by,omitempty"`
	ResolvedAt      *time.Time     `json:"resolved_at,omitempty"`
	ResolvedBy      *uuid.UUID     `json:"resolved_by,omitempty"`
	LastEscalatedAt *time.Time     `json:"last_escalated_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Telegram models

type TelegramUserLink struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	TelegramChatID   *int64     `json:"telegram_chat_id,omitempty"`
	TelegramUsername string     `json:"telegram_username,omitempty"`
	VerificationCode string     `json:"verification_code,omitempty"`
	IsVerified       bool       `json:"is_verified"`
	CreatedAt        time.Time  `json:"created_at"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
}

type TelegramConversation struct {
	ID             uuid.UUID  `json:"id"`
	TelegramChatID int64      `json:"telegram_chat_id"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// DTOs

type CreateChannelRequest struct {
	Name   string                  `json:"name" validate:"required"`
	Type   NotificationChannelType `json:"type" validate:"required"`
	Config json.RawMessage         `json:"config" validate:"required"`
}

type UpdateChannelRequest struct {
	Name     *string          `json:"name,omitempty"`
	Config   *json.RawMessage `json:"config,omitempty"`
	IsActive *bool            `json:"is_active,omitempty"`
}

type CreateAlertRuleRequest struct {
	Name               string        `json:"name" validate:"required"`
	NodeID             uuid.UUID     `json:"node_id" validate:"required"`
	Metric             string        `json:"metric" validate:"required"`
	Operator           string        `json:"operator" validate:"required"`
	Threshold          float64       `json:"threshold"`
	DurationSeconds    int           `json:"duration_seconds"`
	Severity           AlertSeverity `json:"severity" validate:"required"`
	ChannelIDs         []uuid.UUID   `json:"channel_ids"`
	EscalationPolicyID *uuid.UUID    `json:"escalation_policy_id,omitempty"`
	IsActive           *bool         `json:"is_active,omitempty"`
}

type UpdateAlertRuleRequest struct {
	Name               *string        `json:"name,omitempty"`
	Metric             *string        `json:"metric,omitempty"`
	Operator           *string        `json:"operator,omitempty"`
	Threshold          *float64       `json:"threshold,omitempty"`
	DurationSeconds    *int           `json:"duration_seconds,omitempty"`
	Severity           *AlertSeverity `json:"severity,omitempty"`
	ChannelIDs         *[]uuid.UUID   `json:"channel_ids,omitempty"`
	EscalationPolicyID *uuid.UUID     `json:"escalation_policy_id,omitempty"`
	IsActive           *bool          `json:"is_active,omitempty"`
}

// Escalation DTOs

type CreateEscalationPolicyRequest struct {
	Name        string                      `json:"name" validate:"required"`
	Description string                      `json:"description,omitempty"`
	Steps       []CreateEscalationStepInput `json:"steps"`
}

type CreateEscalationStepInput struct {
	StepOrder    int         `json:"step_order"`
	DelaySeconds int         `json:"delay_seconds"`
	ChannelIDs   []uuid.UUID `json:"channel_ids"`
}

type UpdateEscalationPolicyRequest struct {
	Name        *string                     `json:"name,omitempty"`
	Description *string                     `json:"description,omitempty"`
	IsActive    *bool                       `json:"is_active,omitempty"`
	Steps       []CreateEscalationStepInput `json:"steps,omitempty"`
}

// Telegram DTOs

type TelegramLinkRequest struct {
	VerificationCode string `json:"verification_code,omitempty"`
}

type TelegramLinkResponse struct {
	VerificationCode string `json:"verification_code"`
	BotUsername      string `json:"bot_username"`
	IsVerified       bool   `json:"is_verified"`
}
