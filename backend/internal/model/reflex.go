package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ReflexActionType string

const (
	ReflexActionRestartService ReflexActionType = "restart_service"
	ReflexActionClearCache     ReflexActionType = "clear_cache"
	ReflexActionNotify         ReflexActionType = "notify"
	ReflexActionRunCommand     ReflexActionType = "run_command"
	ReflexActionStartVM        ReflexActionType = "start_vm"
	ReflexActionStopVM         ReflexActionType = "stop_vm"
)

func (t ReflexActionType) IsValid() bool {
	switch t {
	case ReflexActionRestartService, ReflexActionClearCache, ReflexActionNotify,
		ReflexActionRunCommand, ReflexActionStartVM, ReflexActionStopVM:
		return true
	}
	return false
}

type ReflexRule struct {
	ID              uuid.UUID        `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description,omitempty"`
	TriggerMetric   string           `json:"trigger_metric"`
	Operator        string           `json:"operator"`
	Threshold       float64          `json:"threshold"`
	ActionType      ReflexActionType `json:"action_type"`
	ActionConfig    json.RawMessage  `json:"action_config"`
	CooldownSeconds int              `json:"cooldown_seconds"`
	IsActive        bool             `json:"is_active"`
	NodeID          *uuid.UUID       `json:"node_id,omitempty"`
	LastTriggeredAt *time.Time       `json:"last_triggered_at,omitempty"`
	TriggerCount    int              `json:"trigger_count"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type CreateReflexRuleRequest struct {
	Name            string           `json:"name" validate:"required"`
	Description     string           `json:"description,omitempty"`
	TriggerMetric   string           `json:"trigger_metric" validate:"required"`
	Operator        string           `json:"operator" validate:"required"`
	Threshold       float64          `json:"threshold"`
	ActionType      ReflexActionType `json:"action_type" validate:"required"`
	ActionConfig    json.RawMessage  `json:"action_config"`
	CooldownSeconds int              `json:"cooldown_seconds"`
	IsActive        *bool            `json:"is_active,omitempty"`
	NodeID          *uuid.UUID       `json:"node_id,omitempty"`
}

type UpdateReflexRuleRequest struct {
	Name            *string           `json:"name,omitempty"`
	Description     *string           `json:"description,omitempty"`
	TriggerMetric   *string           `json:"trigger_metric,omitempty"`
	Operator        *string           `json:"operator,omitempty"`
	Threshold       *float64          `json:"threshold,omitempty"`
	ActionType      *ReflexActionType `json:"action_type,omitempty"`
	ActionConfig    *json.RawMessage  `json:"action_config,omitempty"`
	CooldownSeconds *int              `json:"cooldown_seconds,omitempty"`
	IsActive        *bool             `json:"is_active,omitempty"`
	NodeID          *uuid.UUID        `json:"node_id,omitempty"`
}
