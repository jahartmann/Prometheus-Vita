package handler

import (
	"errors"
	"log/slog"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/notification"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type NotificationHandler struct {
	service  *notification.Service
	alertSvc *notification.AlertService
}

func NewNotificationHandler(service *notification.Service, alertSvc *notification.AlertService) *NotificationHandler {
	return &NotificationHandler{
		service:  service,
		alertSvc: alertSvc,
	}
}

// Channel handlers

func (h *NotificationHandler) ListChannels(c echo.Context) error {
	channels, err := h.service.ListChannels(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list channels")
	}
	return apiPkg.Success(c, channels)
}

func (h *NotificationHandler) CreateChannel(c echo.Context) error {
	var req model.CreateChannelRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" {
		return apiPkg.BadRequest(c, "name is required")
	}
	if !req.Type.IsValid() {
		return apiPkg.BadRequest(c, "type must be 'email', 'telegram', or 'webhook'")
	}

	userID, _ := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	var createdBy *uuid.UUID
	if userID != uuid.Nil {
		createdBy = &userID
	}

	channel, err := h.service.CreateChannel(c.Request().Context(), req, createdBy)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create channel")
	}
	return apiPkg.Created(c, channel)
}

func (h *NotificationHandler) GetChannel(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid channel id")
	}

	channel, err := h.service.GetChannel(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "channel not found")
		}
		return apiPkg.InternalError(c, "failed to get channel")
	}
	return apiPkg.Success(c, channel)
}

func (h *NotificationHandler) UpdateChannel(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid channel id")
	}

	var req model.UpdateChannelRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	channel, err := h.service.UpdateChannel(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "channel not found")
		}
		return apiPkg.InternalError(c, "failed to update channel")
	}
	return apiPkg.Success(c, channel)
}

func (h *NotificationHandler) DeleteChannel(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid channel id")
	}

	if err := h.service.DeleteChannel(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "channel not found")
		}
		return apiPkg.InternalError(c, "failed to delete channel")
	}
	return apiPkg.NoContent(c)
}

func (h *NotificationHandler) TestChannel(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid channel id")
	}

	if err := h.service.TestChannel(c.Request().Context(), id, "Dies ist eine Test-Benachrichtigung von Prometheus."); err != nil {
		slog.Error("notification channel test failed", slog.String("channel_id", id.String()), slog.Any("error", err))
		return apiPkg.BadRequest(c, "test failed")
	}
	return apiPkg.Success(c, map[string]string{"message": "test notification sent"})
}

// History handlers

func (h *NotificationHandler) ListHistory(c echo.Context) error {
	limit, offset := ParsePagination(c)

	entries, err := h.service.ListHistory(c.Request().Context(), limit, offset)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list notification history")
	}
	return apiPkg.Success(c, entries)
}

// Alert rule handlers

func (h *NotificationHandler) ListAlertRules(c echo.Context) error {
	rules, err := h.alertSvc.ListRules(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list alert rules")
	}
	return apiPkg.Success(c, rules)
}

func (h *NotificationHandler) CreateAlertRule(c echo.Context) error {
	var req model.CreateAlertRuleRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" || req.Metric == "" || req.Operator == "" {
		return apiPkg.BadRequest(c, "name, metric, and operator are required")
	}
	if !req.Severity.IsValid() {
		return apiPkg.BadRequest(c, "severity must be 'info', 'warning', or 'critical'")
	}

	rule, err := h.alertSvc.CreateRule(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create alert rule")
	}
	return apiPkg.Created(c, rule)
}

func (h *NotificationHandler) GetAlertRule(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid rule id")
	}

	rule, err := h.alertSvc.GetRule(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "alert rule not found")
		}
		return apiPkg.InternalError(c, "failed to get alert rule")
	}
	return apiPkg.Success(c, rule)
}

func (h *NotificationHandler) UpdateAlertRule(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid rule id")
	}

	var req model.UpdateAlertRuleRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	rule, err := h.alertSvc.UpdateRule(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "alert rule not found")
		}
		return apiPkg.InternalError(c, "failed to update alert rule")
	}
	return apiPkg.Success(c, rule)
}

func (h *NotificationHandler) DeleteAlertRule(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid rule id")
	}

	if err := h.alertSvc.DeleteRule(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "alert rule not found")
		}
		return apiPkg.InternalError(c, "failed to delete alert rule")
	}
	return apiPkg.NoContent(c)
}
