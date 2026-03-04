package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MorningBriefing struct {
	ID          uuid.UUID       `json:"id"`
	Summary     string          `json:"summary"`
	Data        json.RawMessage `json:"data"`
	GeneratedAt time.Time       `json:"generated_at"`
}

type BriefingData struct {
	TotalNodes       int                     `json:"total_nodes"`
	OnlineNodes      int                     `json:"online_nodes"`
	OfflineNodes     int                     `json:"offline_nodes"`
	ActiveAlerts     int                     `json:"active_alerts"`
	UnresolvedAnomalies int                  `json:"unresolved_anomalies"`
	CriticalPredictions int                  `json:"critical_predictions"`
	NodeSummaries    []BriefingNodeSummary   `json:"node_summaries"`
}

type BriefingNodeSummary struct {
	NodeID   string  `json:"node_id"`
	NodeName string  `json:"node_name"`
	IsOnline bool    `json:"is_online"`
	CPUAvg   float64 `json:"cpu_avg"`
	MemPct   float64 `json:"mem_pct"`
	DiskPct  float64 `json:"disk_pct"`
}
