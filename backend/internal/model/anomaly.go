package model

import (
	"time"

	"github.com/google/uuid"
)

type AnomalyRecord struct {
	ID         uuid.UUID  `json:"id"`
	NodeID     uuid.UUID  `json:"node_id"`
	Metric     string     `json:"metric"`
	Value      float64    `json:"value"`
	ZScore     float64    `json:"z_score"`
	Mean       float64    `json:"mean"`
	StdDev     float64    `json:"stddev"`
	Severity   string     `json:"severity"`
	IsResolved bool       `json:"is_resolved"`
	DetectedAt time.Time  `json:"detected_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}
