package handler

import (
	"errors"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ScanBaselineHandler struct {
	repo repository.ScanBaselineRepository
}

func NewScanBaselineHandler(repo repository.ScanBaselineRepository) *ScanBaselineHandler {
	return &ScanBaselineHandler{repo: repo}
}

func (h *ScanBaselineHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}
	baselines, err := h.repo.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return response.InternalError(c, "failed to list baselines")
	}
	if baselines == nil {
		baselines = []model.ScanBaseline{}
	}
	return response.Success(c, baselines)
}

func (h *ScanBaselineHandler) Create(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}

	var req model.CreateBaselineRequest
	// Bind is optional — body may be empty.
	_ = c.Bind(&req)

	baseline := &model.ScanBaseline{
		NodeID:        nodeID,
		Label:         req.Label,
		WhitelistJSON: req.WhitelistJSON,
	}
	if err := h.repo.Create(c.Request().Context(), baseline); err != nil {
		return response.InternalError(c, "failed to create baseline")
	}
	return response.Created(c, baseline)
}

func (h *ScanBaselineHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}

	var req model.UpdateBaselineRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if err := h.repo.Update(c.Request().Context(), id, req); err != nil {
		return response.InternalError(c, "failed to update baseline")
	}
	return response.Success(c, map[string]string{"status": "updated"})
}

func (h *ScanBaselineHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}
	if err := h.repo.Delete(c.Request().Context(), id); err != nil {
		return response.InternalError(c, "failed to delete baseline")
	}
	return response.NoContent(c)
}

func (h *ScanBaselineHandler) Activate(c echo.Context) error {
	baselineID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid baseline id")
	}

	// Fetch baselines for the node by looking up the baseline to get its node_id.
	// We need to find the baseline to get nodeID. List all baselines for any node is not
	// directly available, so we use GetActive or a workaround: get the baseline details
	// from the node param if provided, or list by scanning.
	// The repo Activate requires (nodeID, baselineID). We need nodeID from the baseline.
	// Since there is no GetByID on ScanBaselineRepository, we use a node_id query param or
	// parse it from the URL context. Per the spec: "get nodeID from baseline".
	// We will require a node_id query param to activate.
	nodeIDStr := c.QueryParam("node_id")
	if nodeIDStr == "" {
		// Try path param fallback.
		nodeIDStr = c.Param("node_id")
	}

	var nodeID uuid.UUID
	if nodeIDStr != "" {
		nodeID, err = uuid.Parse(nodeIDStr)
		if err != nil {
			return response.BadRequest(c, "invalid node_id")
		}
	} else {
		// Derive nodeID by listing baselines from all nodes is not possible without GetByID.
		// Use GetActive approach — but we can't look up by baseline ID directly.
		// Fall back: require node_id query param.
		return response.BadRequest(c, "node_id query param required")
	}

	if err := h.repo.Activate(c.Request().Context(), nodeID, baselineID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "baseline not found")
		}
		return response.InternalError(c, "failed to activate baseline")
	}
	return response.NoContent(c)
}
