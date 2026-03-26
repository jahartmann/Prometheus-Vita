package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/updates"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type UpdateHandler struct {
	updateSvc *updates.Service
}

func NewUpdateHandler(updateSvc *updates.Service) *UpdateHandler {
	return &UpdateHandler{updateSvc: updateSvc}
}

func (h *UpdateHandler) ListAll(c echo.Context) error {
	limit, _ := ParsePagination(c)
	checks, err := h.updateSvc.ListAll(c.Request().Context(), limit)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list update checks")
	}
	return apiPkg.Success(c, checks)
}

func (h *UpdateHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	limit, _ := ParsePagination(c)
	checks, err := h.updateSvc.ListByNode(c.Request().Context(), nodeID, limit)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list update checks")
	}
	return apiPkg.Success(c, checks)
}

func (h *UpdateHandler) TriggerCheck(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	check, err := h.updateSvc.CheckUpdates(c.Request().Context(), nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to check updates")
	}
	return apiPkg.Success(c, check)
}
