package model

import (
	"time"

	"github.com/google/uuid"
)

type RecommendationType string

const (
	RecommendationDownsize RecommendationType = "downsize"
	RecommendationUpsize   RecommendationType = "upsize"
	RecommendationOptimal  RecommendationType = "optimal"
)

type ResourceRecommendation struct {
	ID                 uuid.UUID          `json:"id"`
	NodeID             uuid.UUID          `json:"node_id"`
	VMID               int                `json:"vmid"`
	VMName             string             `json:"vm_name"`
	VMType             string             `json:"vm_type"`
	ResourceType       string             `json:"resource_type"` // cpu, memory, disk
	CurrentValue       int64              `json:"current_value"`
	RecommendedValue   int64              `json:"recommended_value"`
	AvgUsage           float64            `json:"avg_usage"`
	MaxUsage           float64            `json:"max_usage"`
	RecommendationType RecommendationType `json:"recommendation_type"`
	Reason             string             `json:"reason,omitempty"`
	VMContext          string             `json:"vm_context,omitempty"`
	ContextReason      string             `json:"context_reason,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
}
