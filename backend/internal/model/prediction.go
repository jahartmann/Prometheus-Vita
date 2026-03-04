package model

import (
	"time"

	"github.com/google/uuid"
)

type MaintenancePrediction struct {
	ID                 uuid.UUID `json:"id"`
	NodeID             uuid.UUID `json:"node_id"`
	Metric             string    `json:"metric"`
	CurrentValue       float64   `json:"current_value"`
	PredictedValue     float64   `json:"predicted_value"`
	Threshold          float64   `json:"threshold"`
	DaysUntilThreshold *float64  `json:"days_until_threshold,omitempty"`
	Slope              float64   `json:"slope"`
	Intercept          float64   `json:"intercept"`
	RSquared           float64   `json:"r_squared"`
	Severity           string    `json:"severity"`
	PredictedAt        time.Time `json:"predicted_at"`
}
