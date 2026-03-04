package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/reflex"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ReflexHandler struct {
	service *reflex.Service
}

func NewReflexHandler(service *reflex.Service) *ReflexHandler {
	return &ReflexHandler{service: service}
}

func (h *ReflexHandler) List(c echo.Context) error {
	rules, err := h.service.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Reflex-Regeln")
	}
	if rules == nil {
		rules = []model.ReflexRule{}
	}
	return apiPkg.Success(c, rules)
}

func (h *ReflexHandler) Create(c echo.Context) error {
	var req model.CreateReflexRuleRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" || req.TriggerMetric == "" || req.Operator == "" {
		return apiPkg.BadRequest(c, "name, trigger_metric, and operator are required")
	}
	if !req.ActionType.IsValid() {
		return apiPkg.BadRequest(c, "invalid action_type")
	}

	rule, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Erstellen der Reflex-Regel")
	}
	return apiPkg.Created(c, rule)
}

func (h *ReflexHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Reflex-ID")
	}

	rule, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "Reflex-Regel nicht gefunden")
		}
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Reflex-Regel")
	}
	return apiPkg.Success(c, rule)
}

func (h *ReflexHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Reflex-ID")
	}

	var req model.UpdateReflexRuleRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	rule, err := h.service.Update(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "Reflex-Regel nicht gefunden")
		}
		return apiPkg.InternalError(c, "Fehler beim Aktualisieren der Reflex-Regel")
	}
	return apiPkg.Success(c, rule)
}

func (h *ReflexHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Reflex-ID")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "Reflex-Regel nicht gefunden")
		}
		return apiPkg.InternalError(c, "Fehler beim Loeschen der Reflex-Regel")
	}
	return apiPkg.NoContent(c)
}
