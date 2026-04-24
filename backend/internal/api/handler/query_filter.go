package handler

import (
	"strconv"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func ParseQueryFilter(c echo.Context) model.QueryFilter {
	limit, offset := ParsePagination(c)
	filter := model.QueryFilter{
		Limit:    limit,
		Offset:   offset,
		Source:   c.QueryParam("source"),
		Status:   c.QueryParam("status"),
		Severity: c.QueryParam("severity"),
		Category: c.QueryParam("category"),
		Type:     c.QueryParam("type"),
		Method:   c.QueryParam("method"),
		Query:    c.QueryParam("q"),
	}
	if raw := c.QueryParam("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			filter.Limit = parsed
		}
	}
	if raw := c.QueryParam("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			filter.Offset = parsed
		}
	}
	if raw := c.QueryParam("node_id"); raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil {
			filter.NodeID = &parsed
		}
	}
	if raw := c.QueryParam("user_id"); raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil {
			filter.UserID = &parsed
		}
	}
	if raw := c.QueryParam("from"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.From = &parsed
		}
	}
	if raw := c.QueryParam("to"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.To = &parsed
		}
	}
	return filter
}
