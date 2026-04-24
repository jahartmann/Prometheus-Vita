package handler

import (
	"strconv"
	"time"

	"github.com/antigravity/prometheus/internal/api/middleware"
	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/service/operations"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type OperationsHandler struct {
	service *operations.Service
}

func NewOperationsHandler(service *operations.Service) *OperationsHandler {
	return &OperationsHandler{service: service}
}

func (h *OperationsHandler) ListTasks(c echo.Context) error {
	tasks, err := h.service.ListTasks(c.Request().Context(), parseOperationsQuery(c))
	if err != nil {
		return apiPkg.InternalError(c, "failed to list tasks")
	}
	return apiPkg.Success(c, tasks)
}

func (h *OperationsHandler) Timeline(c echo.Context) error {
	events, err := h.service.Timeline(c.Request().Context(), parseOperationsQuery(c))
	if err != nil {
		return apiPkg.InternalError(c, "failed to build timeline")
	}
	return apiPkg.Success(c, events)
}

func (h *OperationsHandler) AnalyzeRCA(c echo.Context) error {
	var req model.RCAAnalyzeRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	resp, err := h.service.AnalyzeRCA(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to analyze root cause")
	}
	return apiPkg.Success(c, resp)
}

func (h *OperationsHandler) KnowledgeGraph(c echo.Context) error {
	graph, err := h.service.KnowledgeGraph(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to build knowledge graph")
	}
	return apiPkg.Success(c, graph)
}

func (h *OperationsHandler) GenerateReport(c echo.Context) error {
	var req model.OperationsReportRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	resp, err := h.service.GenerateReport(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to generate report")
	}
	return apiPkg.Success(c, resp)
}

func parseOperationsQuery(c echo.Context) operations.Query {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	var nodeID *uuid.UUID
	if raw := c.QueryParam("node_id"); raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil {
			nodeID = &parsed
		}
	}
	var userID *uuid.UUID
	if raw, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID); ok {
		userID = &raw
	}
	var from *time.Time
	if raw := c.QueryParam("from"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			from = &parsed
		}
	}
	var to *time.Time
	if raw := c.QueryParam("to"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			to = &parsed
		}
	}
	return operations.Query{
		Limit: limit,
		Source: c.QueryParam("source"),
		Severity: c.QueryParam("severity"),
		Status: c.QueryParam("status"),
		NodeID: nodeID,
		UserID: userID,
		From: from,
		To: to,
		Query: c.QueryParam("q"),
	}
}
