package handler

import (
	"errors"
	"strconv"

	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/notification"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type EscalationHandler struct {
	service *notification.EscalationService
}

func NewEscalationHandler(service *notification.EscalationService) *EscalationHandler {
	return &EscalationHandler{service: service}
}

// Policy CRUD

func (h *EscalationHandler) ListPolicies(c echo.Context) error {
	policies, err := h.service.ListPolicies(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list escalation policies")
	}
	if policies == nil {
		policies = []model.EscalationPolicy{}
	}
	return apiPkg.Success(c, policies)
}

func (h *EscalationHandler) CreatePolicy(c echo.Context) error {
	var req model.CreateEscalationPolicyRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" {
		return apiPkg.BadRequest(c, "name is required")
	}

	policy, err := h.service.CreatePolicy(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create escalation policy")
	}
	return apiPkg.Created(c, policy)
}

func (h *EscalationHandler) GetPolicy(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid policy id")
	}

	policy, err := h.service.GetPolicy(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "escalation policy not found")
		}
		return apiPkg.InternalError(c, "failed to get escalation policy")
	}
	return apiPkg.Success(c, policy)
}

func (h *EscalationHandler) UpdatePolicy(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid policy id")
	}

	var req model.UpdateEscalationPolicyRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	policy, err := h.service.UpdatePolicy(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "escalation policy not found")
		}
		return apiPkg.InternalError(c, "failed to update escalation policy")
	}
	return apiPkg.Success(c, policy)
}

func (h *EscalationHandler) DeletePolicy(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid policy id")
	}

	if err := h.service.DeletePolicy(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "escalation policy not found")
		}
		return apiPkg.InternalError(c, "failed to delete escalation policy")
	}
	return apiPkg.NoContent(c)
}

// Incident management

func (h *EscalationHandler) ListIncidents(c echo.Context) error {
	limit := 50
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	incidents, err := h.service.ListIncidents(c.Request().Context(), limit, offset)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list incidents")
	}
	if incidents == nil {
		incidents = []model.AlertIncident{}
	}
	return apiPkg.Success(c, incidents)
}

func (h *EscalationHandler) GetIncident(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid incident id")
	}

	incident, err := h.service.GetIncident(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "incident not found")
		}
		return apiPkg.InternalError(c, "failed to get incident")
	}
	return apiPkg.Success(c, incident)
}

func (h *EscalationHandler) AcknowledgeIncident(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid incident id")
	}

	userID, _ := c.Get(middleware.ContextKeyUserID).(uuid.UUID)

	if err := h.service.AcknowledgeIncident(c.Request().Context(), id, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "incident not found")
		}
		return apiPkg.BadRequest(c, err.Error())
	}
	return apiPkg.Success(c, map[string]string{"message": "incident acknowledged"})
}

func (h *EscalationHandler) ResolveIncident(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid incident id")
	}

	userID, _ := c.Get(middleware.ContextKeyUserID).(uuid.UUID)

	if err := h.service.ResolveIncident(c.Request().Context(), id, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "incident not found")
		}
		return apiPkg.BadRequest(c, err.Error())
	}
	return apiPkg.Success(c, map[string]string{"message": "incident resolved"})
}
