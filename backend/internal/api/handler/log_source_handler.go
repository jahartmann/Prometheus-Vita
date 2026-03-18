package handler

import (
	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/logscan"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type LogSourceHandler struct {
	sourceRepo   repository.LogSourceRepository
	discoverySvc *logscan.DiscoveryService
}

func NewLogSourceHandler(sourceRepo repository.LogSourceRepository, discoverySvc *logscan.DiscoveryService) *LogSourceHandler {
	return &LogSourceHandler{
		sourceRepo:   sourceRepo,
		discoverySvc: discoverySvc,
	}
}

func (h *LogSourceHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}
	sources, err := h.sourceRepo.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return response.InternalError(c, "failed to list log sources")
	}
	if sources == nil {
		sources = []model.LogSource{}
	}
	return response.Success(c, sources)
}

func (h *LogSourceHandler) Update(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}

	var req model.UpdateLogSourcesRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	// Validate that each source path exists for this node.
	existing, err := h.sourceRepo.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return response.InternalError(c, "failed to validate log sources")
	}
	existingPaths := make(map[string]struct{}, len(existing))
	for _, s := range existing {
		existingPaths[s.Path] = struct{}{}
	}

	for _, s := range req.Sources {
		if _, ok := existingPaths[s.Path]; !ok {
			return response.BadRequest(c, "source path not found: "+s.Path)
		}
	}

	ctx := c.Request().Context()
	for _, s := range req.Sources {
		if err := h.sourceRepo.UpdateEnabled(ctx, nodeID, s.Path, s.Enabled); err != nil {
			return response.InternalError(c, "failed to update log source")
		}
	}

	sources, err := h.sourceRepo.ListByNode(ctx, nodeID)
	if err != nil {
		return response.InternalError(c, "failed to list updated log sources")
	}
	return response.Success(c, sources)
}
