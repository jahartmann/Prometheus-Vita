package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/repository"
)

type GetMetricsTool struct {
	metricsRepo repository.MetricsRepository
}

func NewGetMetricsTool(metricsRepo repository.MetricsRepository) *GetMetricsTool {
	return &GetMetricsTool{metricsRepo: metricsRepo}
}

func (t *GetMetricsTool) Name() string {
	return "get_metrics"
}

func (t *GetMetricsTool) Description() string {
	return "Ruft Metriken (CPU, Memory, Disk, Netzwerk) fuer einen Node ab"
}

func (t *GetMetricsTool) ReadOnly() bool { return true }

func (t *GetMetricsTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"node_id": {
				"type": "string",
				"description": "Die UUID des Nodes"
			},
			"hours": {
				"type": "number",
				"description": "Zeitraum in Stunden (Standard: 1)"
			}
		},
		"required": ["node_id"]
	}`)
}

func (t *GetMetricsTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		NodeID string  `json:"node_id"`
		Hours  float64 `json:"hours"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	nodeID, err := uuid.Parse(params.NodeID)
	if err != nil {
		return json.Marshal(map[string]string{"error": "Ungueltige Node-ID"})
	}

	hours := params.Hours
	if hours <= 0 {
		hours = 1
	}

	now := time.Now()
	since := now.Add(-time.Duration(hours * float64(time.Hour)))

	records, err := t.metricsRepo.GetByNode(ctx, nodeID, since, now)
	if err != nil {
		return json.Marshal(map[string]string{"error": fmt.Sprintf("Fehler beim Abrufen der Metriken: %v", err)})
	}

	if len(records) == 0 {
		return json.Marshal(map[string]string{"message": "Keine Metriken im angegebenen Zeitraum gefunden"})
	}

	type metricsSummary struct {
		RecordCount int     `json:"record_count"`
		AvgCPU      float64 `json:"avg_cpu_percent"`
		MaxCPU      float64 `json:"max_cpu_percent"`
		AvgMemPct   float64 `json:"avg_memory_percent"`
		MaxMemPct   float64 `json:"max_memory_percent"`
		AvgDiskPct  float64 `json:"avg_disk_percent"`
		LatestNetIn int64   `json:"latest_net_in_bytes"`
		LatestNetOut int64  `json:"latest_net_out_bytes"`
	}

	var totalCPU, maxCPU float64
	var totalMemPct, maxMemPct float64
	var totalDiskPct float64

	for _, r := range records {
		totalCPU += r.CPUUsage
		if r.CPUUsage > maxCPU {
			maxCPU = r.CPUUsage
		}
		var memPct float64
		if r.MemTotal > 0 {
			memPct = float64(r.MemUsed) / float64(r.MemTotal) * 100
		}
		totalMemPct += memPct
		if memPct > maxMemPct {
			maxMemPct = memPct
		}
		var diskPct float64
		if r.DiskTotal > 0 {
			diskPct = float64(r.DiskUsed) / float64(r.DiskTotal) * 100
		}
		totalDiskPct += diskPct
	}

	count := float64(len(records))
	latest := records[len(records)-1]

	summary := metricsSummary{
		RecordCount:  len(records),
		AvgCPU:       totalCPU / count,
		MaxCPU:       maxCPU,
		AvgMemPct:    totalMemPct / count,
		MaxMemPct:    maxMemPct,
		AvgDiskPct:   totalDiskPct / count,
		LatestNetIn:  latest.NetIn,
		LatestNetOut: latest.NetOut,
	}

	return json.Marshal(summary)
}
