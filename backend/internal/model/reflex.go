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
	ReflexActionScaleUp        ReflexActionType = "scale_up"
	ReflexActionScaleDown      ReflexActionType = "scale_down"
	ReflexActionSnapshot       ReflexActionType = "snapshot"
	ReflexActionAIAnalyze      ReflexActionType = "ai_analyze"
)

func (t ReflexActionType) IsValid() bool {
	switch t {
	case ReflexActionRestartService, ReflexActionClearCache, ReflexActionNotify,
		ReflexActionRunCommand, ReflexActionStartVM, ReflexActionStopVM,
		ReflexActionScaleUp, ReflexActionScaleDown, ReflexActionSnapshot, ReflexActionAIAnalyze:
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

	// Time-based scheduling
	ScheduleType    string `json:"schedule_type,omitempty"`      // "always", "time_window", "cron"
	ScheduleCron    string `json:"schedule_cron,omitempty"`      // Cron expression for scheduled rules
	TimeWindowStart string `json:"time_window_start,omitempty"`  // "08:00" format
	TimeWindowEnd   string `json:"time_window_end,omitempty"`    // "18:00" format
	TimeWindowDays  []int  `json:"time_window_days,omitempty"`   // 0=Sun, 1=Mon, ..., 6=Sat

	// AI integration
	AIEnabled        bool   `json:"ai_enabled"`                   // Whether AI evaluates this rule
	AISeverity       string `json:"ai_severity,omitempty"`        // AI-assessed severity
	AIRecommendation string `json:"ai_recommendation,omitempty"`  // AI suggestion for this rule

	// Rule chaining
	Priority int      `json:"priority"`        // Execution priority (lower = first)
	Tags     []string `json:"tags,omitempty"`  // Categorization tags

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ScheduleType    string           `json:"schedule_type,omitempty"`
	ScheduleCron    string           `json:"schedule_cron,omitempty"`
	TimeWindowStart string           `json:"time_window_start,omitempty"`
	TimeWindowEnd   string           `json:"time_window_end,omitempty"`
	TimeWindowDays  []int            `json:"time_window_days,omitempty"`
	AIEnabled       *bool            `json:"ai_enabled,omitempty"`
	Priority        *int             `json:"priority,omitempty"`
	Tags            []string         `json:"tags,omitempty"`
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
	ScheduleType    *string           `json:"schedule_type,omitempty"`
	ScheduleCron    *string           `json:"schedule_cron,omitempty"`
	TimeWindowStart *string           `json:"time_window_start,omitempty"`
	TimeWindowEnd   *string           `json:"time_window_end,omitempty"`
	TimeWindowDays  []int             `json:"time_window_days,omitempty"`
	AIEnabled       *bool             `json:"ai_enabled,omitempty"`
	Priority        *int              `json:"priority,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
}
