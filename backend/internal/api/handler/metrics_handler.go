package handler

import (
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
