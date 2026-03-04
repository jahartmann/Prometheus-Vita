package handler

import (
	"errors"
	"strconv"

	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/rightsizing"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type RightsizingHandler struct {
	rightsizingSvc *rightsizing.Service
}

func NewRightsizingHandler(rightsizingSvc *rightsizing.Service) *RightsizingHandler {
	return &RightsizingHandler{rightsizingSvc: rightsizingSvc}
}

func (h *RightsizingHandler) ListAll(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	recs, err := h.rightsizingSvc.ListAll(c.Request().Context(), limit)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list recommendations")
	}
	return apiPkg.Success(c, recs)
}

func (h *RightsizingHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	recs, err := h.rightsizingSvc.ListByNode(c.Request().Context(), nodeID, limit)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list recommendations")
	}
	return apiPkg.Success(c, recs)
}

func (h *RightsizingHandler) TriggerAnalysis(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	recs, err := h.rightsizingSvc.AnalyzeNode(c.Request().Context(), nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to analyze node")
	}
	return apiPkg.Success(c, recs)
}
