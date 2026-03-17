package model

import (
	"time"

	"github.com/google/uuid"
)

type LogReportSchedule struct {
	ID                 uuid.UUID   `json:"id"`
	CronExpression     string      `json:"cron_expression"`
	NodeIDs            []uuid.UUID `json:"node_ids"`
	TimeWindowHours    int         `json:"time_window_hours"`
	DeliveryChannelIDs []uuid.UUID `json:"delivery_channel_ids,omitempty"`
	IsActive           bool        `json:"is_active"`
	LastRunAt          *time.Time  `json:"last_run_at,omitempty"`
	NextRunAt          *time.Time  `json:"next_run_at,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
}

type CreateLogReportScheduleRequest struct {
	CronExpression     string      `json:"cron_expression" validate:"required"`
	NodeIDs            []uuid.UUID `json:"node_ids" validate:"required"`
	TimeWindowHours    int         `json:"time_window_hours"`
	DeliveryChannelIDs []uuid.UUID `json:"delivery_channel_ids,omitempty"`
}

type UpdateLogReportScheduleRequest struct {
	CronExpression     *string      `json:"cron_expression,omitempty"`
	NodeIDs            *[]uuid.UUID `json:"node_ids,omitempty"`
	TimeWindowHours    *int         `json:"time_window_hours,omitempty"`
	DeliveryChannelIDs *[]uuid.UUID `json:"delivery_channel_ids,omitempty"`
	IsActive           *bool        `json:"is_active,omitempty"`
}
