package handler

import (
	"errors"
	"strconv"

	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/drift"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type DriftHandler struct {
	driftSvc *drift.Service
}

func NewDriftHandler(driftSvc *drift.Service) *DriftHandler {
	return &DriftHandler{driftSvc: driftSvc}
}

func (h *DriftHandler) ListAll(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 50
	}
	checks, err := h.driftSvc.ListAll(c.Request().Context(), limit)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list drift checks")
	}
	return apiPkg.Success(c, checks)
}

func (h *DriftHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	checks, err := h.driftSvc.ListByNode(c.Request().Context(), nodeID, limit)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list drift checks")
	}
	return apiPkg.Success(c, checks)
}

func (h *DriftHandler) TriggerCheck(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	check, err := h.driftSvc.CheckDrift(c.Request().Context(), nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to run drift check")
	}
	return apiPkg.Success(c, check)
}
