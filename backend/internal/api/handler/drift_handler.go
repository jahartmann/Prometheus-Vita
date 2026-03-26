package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
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
	limit, _ := ParsePagination(c)
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
	limit, _ := ParsePagination(c)
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

func (h *DriftHandler) AcceptBaseline(c echo.Context) error {
	checkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid drift check id")
	}

	if err := h.driftSvc.AcceptBaseline(c.Request().Context(), checkID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "drift check not found")
		}
		return apiPkg.InternalError(c, "failed to accept baseline")
	}

	return apiPkg.Success(c, map[string]string{"message": "Baseline aktualisiert"})
}

func (h *DriftHandler) IgnoreDrift(c echo.Context) error {
	checkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid drift check id")
	}

	var req struct {
		FilePath string `json:"file_path"`
	}
	if err := c.Bind(&req); err != nil || req.FilePath == "" {
		return apiPkg.BadRequest(c, "file_path is required")
	}

	if err := h.driftSvc.IgnoreDrift(c.Request().Context(), checkID, req.FilePath); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "drift check not found")
		}
		return apiPkg.InternalError(c, "failed to ignore drift")
	}

	return apiPkg.Success(c, map[string]string{"message": "Drift ignoriert"})
}

func (h *DriftHandler) CompareNodes(c echo.Context) error {
	var req model.CompareNodesRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if len(req.NodeIDs) < 2 {
		return apiPkg.BadRequest(c, "at least 2 node IDs required")
	}
	if len(req.FilePaths) == 0 {
		return apiPkg.BadRequest(c, "at least 1 file path required")
	}

	result, err := h.driftSvc.CompareNodes(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to compare nodes: "+err.Error())
	}

	return apiPkg.Success(c, result)
}
