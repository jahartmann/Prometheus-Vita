package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/recovery"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type DRHandler struct {
	profileSvc   *recovery.ProfileService
	readinessSvc *recovery.ReadinessService
	runbookSvc   *recovery.RunbookService
}

func NewDRHandler(
	profileSvc *recovery.ProfileService,
	readinessSvc *recovery.ReadinessService,
	runbookSvc *recovery.RunbookService,
) *DRHandler {
	return &DRHandler{
		profileSvc:   profileSvc,
		readinessSvc: readinessSvc,
		runbookSvc:   runbookSvc,
	}
}

// GetLatestProfile handles GET /nodes/:id/dr/profile
func (h *DRHandler) GetLatestProfile(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	profile, err := h.profileSvc.GetLatestProfile(c.Request().Context(), nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "no profile found for this node")
		}
		return apiPkg.InternalError(c, "failed to get profile")
	}

	return apiPkg.Success(c, profile)
}

// CollectProfile handles POST /nodes/:id/dr/profile
func (h *DRHandler) CollectProfile(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	profile, err := h.profileSvc.CollectProfile(c.Request().Context(), nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to collect profile")
	}

	return apiPkg.Created(c, profile)
}

// GetReadiness handles GET /nodes/:id/dr/readiness
func (h *DRHandler) GetReadiness(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	score, err := h.readinessSvc.GetLatestScore(c.Request().Context(), nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "no readiness score found for this node")
		}
		return apiPkg.InternalError(c, "failed to get readiness score")
	}

	return apiPkg.Success(c, score)
}

// CalculateReadiness handles POST /nodes/:id/dr/readiness
func (h *DRHandler) CalculateReadiness(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	score, err := h.readinessSvc.CalculateScore(c.Request().Context(), nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to calculate readiness score")
	}

	return apiPkg.Created(c, score)
}

// ListRunbooks handles GET /nodes/:id/dr/runbooks
func (h *DRHandler) ListRunbooks(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	runbooks, err := h.runbookSvc.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list runbooks")
	}

	return apiPkg.Success(c, runbooks)
}

// GenerateRunbook handles POST /nodes/:id/dr/runbooks
func (h *DRHandler) GenerateRunbook(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	var req model.GenerateRunbookRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if req.Scenario == "" {
		return apiPkg.BadRequest(c, "scenario is required")
	}

	runbook, err := h.runbookSvc.GenerateRunbook(c.Request().Context(), nodeID, req.Scenario)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to generate runbook")
	}

	return apiPkg.Created(c, runbook)
}

// GetRunbook handles GET /dr/runbooks/:id
func (h *DRHandler) GetRunbook(c echo.Context) error {
	runbookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid runbook id")
	}

	runbook, err := h.runbookSvc.GetRunbook(c.Request().Context(), runbookID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "runbook not found")
		}
		return apiPkg.InternalError(c, "failed to get runbook")
	}

	return apiPkg.Success(c, runbook)
}

// UpdateRunbook handles PUT /dr/runbooks/:id
func (h *DRHandler) UpdateRunbook(c echo.Context) error {
	runbookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid runbook id")
	}

	var req model.UpdateRunbookRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	runbook, err := h.runbookSvc.UpdateRunbook(c.Request().Context(), runbookID, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "runbook not found")
		}
		return apiPkg.InternalError(c, "failed to update runbook")
	}

	return apiPkg.Success(c, runbook)
}

// DeleteRunbook handles DELETE /dr/runbooks/:id
func (h *DRHandler) DeleteRunbook(c echo.Context) error {
	runbookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid runbook id")
	}

	if err := h.runbookSvc.DeleteRunbook(c.Request().Context(), runbookID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "runbook not found")
		}
		return apiPkg.InternalError(c, "failed to delete runbook")
	}

	return apiPkg.NoContent(c)
}

// SimulateDR handles POST /dr/simulate
func (h *DRHandler) SimulateDR(c echo.Context) error {
	var req model.DRSimulationRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if req.NodeID == uuid.Nil {
		return apiPkg.BadRequest(c, "node_id is required")
	}
	if req.Scenario == "" {
		return apiPkg.BadRequest(c, "scenario is required")
	}

	result, err := h.runbookSvc.SimulateDR(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to simulate DR")
	}

	return apiPkg.Success(c, result)
}

// ListAllScores handles GET /dr/scores
func (h *DRHandler) ListAllScores(c echo.Context) error {
	scores, err := h.readinessSvc.ListAllScores(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list readiness scores")
	}

	return apiPkg.Success(c, scores)
}
