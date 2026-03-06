package handler

import (
	"time"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/labstack/echo/v4"
)

type ClusterHandler struct {
	monitorSvc *monitor.Service
}

func NewClusterHandler(monitorSvc *monitor.Service) *ClusterHandler {
	return &ClusterHandler{monitorSvc: monitorSvc}
}

type ClusterSummary struct {
	TotalNodes   int                          `json:"total_nodes"`
	OnlineNodes  int                          `json:"online_nodes"`
	TotalCPU     float64                      `json:"total_cpu_usage"`
	TotalMemUsed int64                        `json:"total_mem_used"`
	TotalMemMax  int64                        `json:"total_mem_total"`
	TotalDiskUsed int64                       `json:"total_disk_used"`
	TotalDiskMax  int64                       `json:"total_disk_total"`
	Nodes        []monitor.NodeStatusSummary  `json:"nodes"`
}

func (h *ClusterHandler) GetSummary(c echo.Context) error {
	nodes, err := h.monitorSvc.GetAllNodesStatus(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Cluster-Daten")
	}

	summary := ClusterSummary{
		TotalNodes: len(nodes),
		Nodes:      nodes,
	}

	for _, n := range nodes {
		if n.IsOnline {
			summary.OnlineNodes++
		}
		if n.Status != nil {
			summary.TotalCPU += n.Status.CPUUsage
			summary.TotalMemUsed += n.Status.MemUsed
			summary.TotalMemMax += n.Status.MemTotal
			summary.TotalDiskUsed += n.Status.DiskUsed
			summary.TotalDiskMax += n.Status.DiskTotal
		}
	}

	return apiPkg.Success(c, summary)
}

type ClusterHistoryPoint struct {
	Time     time.Time `json:"time"`
	CPUAvg   float64   `json:"cpu_avg"`
	MemPct   float64   `json:"mem_pct"`
	DiskPct  float64   `json:"disk_pct"`
	NetIn    int64     `json:"net_in"`
	NetOut   int64     `json:"net_out"`
}

func (h *ClusterHandler) GetHistory(c echo.Context) error {
	period := c.QueryParam("period")
	if period == "" {
		period = "24h"
	}

	hours := 24
	switch period {
	case "1h":
		hours = 1
	case "6h":
		hours = 6
	case "24h":
		hours = 24
	case "7d":
		hours = 168
	case "30d":
		hours = 720
	}

	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	until := time.Now().UTC()

	history, err := h.monitorSvc.GetClusterHistory(c.Request().Context(), since, until)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Cluster-Historie")
	}

	return apiPkg.Success(c, history)
}
