package handler

import (
	"strconv"
	"time"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type MetricsHandler struct {
	monitorSvc *monitor.Service
}

func NewMetricsHandler(monitorSvc *monitor.Service) *MetricsHandler {
	return &MetricsHandler{monitorSvc: monitorSvc}
}

func (h *MetricsHandler) GetMetricsHistory(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	now := time.Now().UTC()
	since := now.Add(-1 * time.Hour)
	until := now

	if s := c.QueryParam("since"); s != "" {
		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return apiPkg.BadRequest(c, "invalid 'since' parameter, must be RFC3339 format")
		}
		since = parsed
	}

	if u := c.QueryParam("until"); u != "" {
		parsed, err := time.Parse(time.RFC3339, u)
		if err != nil {
			return apiPkg.BadRequest(c, "invalid 'until' parameter, must be RFC3339 format")
		}
		until = parsed
	}

	// Validate time range
	if !since.Before(until) {
		return apiPkg.BadRequest(c, "'since' must be before 'until'")
	}
	maxRange := 90 * 24 * time.Hour
	if until.Sub(since) > maxRange {
		return apiPkg.BadRequest(c, "time range must not exceed 90 days")
	}

	records, err := h.monitorSvc.GetMetricsHistory(c.Request().Context(), id, since, until)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get metrics history")
	}

	return apiPkg.Success(c, records)
}

func (h *MetricsHandler) GetMetricsSummary(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	period := c.QueryParam("period")
	if period == "" {
		period = "24h"
	}

	var duration time.Duration
	switch period {
	case "1h":
		duration = 1 * time.Hour
	case "6h":
		duration = 6 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		return apiPkg.BadRequest(c, "invalid period, must be one of: 1h, 6h, 24h, 7d, 30d")
	}

	now := time.Now().UTC()
	since := now.Add(-duration)

	summary, err := h.monitorSvc.GetMetricsSummary(c.Request().Context(), id, since, now)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get metrics summary")
	}

	summary.Period = period
	return apiPkg.Success(c, summary)
}

// parsePeriod converts a period string to a time range (start, end).
func parsePeriod(period string) (time.Time, time.Time, bool) {
	now := time.Now().UTC()
	var duration time.Duration
	switch period {
	case "1h":
		duration = 1 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	case "all":
		// Use 1 year as "all"
		duration = 365 * 24 * time.Hour
	default:
		return time.Time{}, time.Time{}, false
	}
	return now.Add(-duration), now, true
}

// GetVMMetricsHistory returns metrics history for a specific VM.
func (h *MetricsHandler) GetVMMetricsHistory(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now

	if s := c.QueryParam("start"); s != "" {
		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return apiPkg.BadRequest(c, "invalid 'start' parameter")
		}
		start = parsed
	}
	if e := c.QueryParam("end"); e != "" {
		parsed, err := time.Parse(time.RFC3339, e)
		if err != nil {
			return apiPkg.BadRequest(c, "invalid 'end' parameter")
		}
		end = parsed
	}

	records, err := h.monitorSvc.GetVMMetricsHistory(c.Request().Context(), nodeID, vmid, start, end)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get vm metrics history")
	}

	return apiPkg.Success(c, records)
}

// GetVMNetworkSummary returns network summary for a specific VM.
func (h *MetricsHandler) GetVMNetworkSummary(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	period := c.QueryParam("period")
	if period == "" {
		period = "24h"
	}

	start, end, valid := parsePeriod(period)
	if !valid {
		return apiPkg.BadRequest(c, "invalid period, must be one of: 1h, 24h, 7d, 30d, all")
	}

	summary, err := h.monitorSvc.GetVMNetworkSummary(c.Request().Context(), nodeID, vmid, start, end)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get vm network summary")
	}

	return apiPkg.Success(c, summary)
}

// GetNodeNetworkSummary returns aggregated network summary for a node.
func (h *MetricsHandler) GetNodeNetworkSummary(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	period := c.QueryParam("period")
	if period == "" {
		period = "24h"
	}

	start, end, valid := parsePeriod(period)
	if !valid {
		return apiPkg.BadRequest(c, "invalid period, must be one of: 1h, 24h, 7d, 30d, all")
	}

	summary, err := h.monitorSvc.GetNodeNetworkSummary(c.Request().Context(), nodeID, start, end)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get node network summary")
	}

	return apiPkg.Success(c, summary)
}

// GetClusterNetworkSummary returns cluster-wide network summary.
func (h *MetricsHandler) GetClusterNetworkSummary(c echo.Context) error {
	period := c.QueryParam("period")
	if period == "" {
		period = "24h"
	}

	start, end, valid := parsePeriod(period)
	if !valid {
		return apiPkg.BadRequest(c, "invalid period, must be one of: 1h, 24h, 7d, 30d, all")
	}

	summary, err := h.monitorSvc.GetClusterNetworkSummary(c.Request().Context(), start, end)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get cluster network summary")
	}

	return apiPkg.Success(c, summary)
}
