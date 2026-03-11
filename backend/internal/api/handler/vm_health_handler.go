package handler

import (
	"strconv"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	vmService "github.com/antigravity/prometheus/internal/service/vm"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type VMHealthHandler struct {
	healthSvc      *vmService.HealthService
	rightsizingSvc *vmService.RightsizingService
	anomalySvc     *vmService.AnomalyService
	snapshotSvc    *vmService.SnapshotPolicyService
	scheduledSvc   *vmService.ScheduledActionService
	depSvc         *vmService.DependencyService
}

func NewVMHealthHandler(
	healthSvc *vmService.HealthService,
	rightsizingSvc *vmService.RightsizingService,
	anomalySvc *vmService.AnomalyService,
	snapshotSvc *vmService.SnapshotPolicyService,
	scheduledSvc *vmService.ScheduledActionService,
	depSvc *vmService.DependencyService,
) *VMHealthHandler {
	return &VMHealthHandler{
		healthSvc:      healthSvc,
		rightsizingSvc: rightsizingSvc,
		anomalySvc:     anomalySvc,
		snapshotSvc:    snapshotSvc,
		scheduledSvc:   scheduledSvc,
		depSvc:         depSvc,
	}
}

// --- Health Score ---

func (h *VMHealthHandler) GetVMHealth(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	score, err := h.healthSvc.CalculateHealthScore(c.Request().Context(), nodeID, vmid)
	if err != nil {
		return apiPkg.InternalError(c, "failed to calculate health score")
	}
	return apiPkg.Success(c, score)
}

func (h *VMHealthHandler) GetAllVMHealth(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	scores, err := h.healthSvc.CalculateAllHealthScores(c.Request().Context(), nodeID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to calculate health scores")
	}
	if scores == nil {
		scores = []model.HealthScore{}
	}
	return apiPkg.Success(c, scores)
}

// --- VM-level Rightsizing ---

func (h *VMHealthHandler) GetVMRightsizing(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	rec, err := h.rightsizingSvc.AnalyzeVM(c.Request().Context(), nodeID, vmid)
	if err != nil {
		return apiPkg.InternalError(c, "failed to analyze VM")
	}
	return apiPkg.Success(c, rec)
}

// --- VM-level Anomalies ---

func (h *VMHealthHandler) GetVMAnomalies(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	anomalies, err := h.anomalySvc.DetectAnomalies(c.Request().Context(), nodeID, vmid)
	if err != nil {
		return apiPkg.InternalError(c, "failed to detect anomalies")
	}
	if anomalies == nil {
		anomalies = []model.VMAnomaly{}
	}
	return apiPkg.Success(c, anomalies)
}

// --- Snapshot Policies ---

func (h *VMHealthHandler) ListSnapshotPolicies(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	policies, err := h.snapshotSvc.ListByVM(c.Request().Context(), nodeID, vmid)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list snapshot policies")
	}
	if policies == nil {
		policies = []model.SnapshotPolicy{}
	}
	return apiPkg.Success(c, policies)
}

func (h *VMHealthHandler) CreateSnapshotPolicy(c echo.Context) error {
	var req model.CreateSnapshotPolicyRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	policy, err := h.snapshotSvc.Create(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create snapshot policy")
	}
	return apiPkg.Created(c, policy)
}

func (h *VMHealthHandler) UpdateSnapshotPolicy(c echo.Context) error {
	id, err := uuid.Parse(c.Param("policyId"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid policy id")
	}

	var req model.UpdateSnapshotPolicyRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	policy, err := h.snapshotSvc.Update(c.Request().Context(), id, req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to update snapshot policy")
	}
	return apiPkg.Success(c, policy)
}

func (h *VMHealthHandler) DeleteSnapshotPolicy(c echo.Context) error {
	id, err := uuid.Parse(c.Param("policyId"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid policy id")
	}

	if err := h.snapshotSvc.Delete(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "failed to delete snapshot policy")
	}
	return apiPkg.NoContent(c)
}

// --- Scheduled Actions ---

func (h *VMHealthHandler) ListScheduledActions(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	actions, err := h.scheduledSvc.ListByVM(c.Request().Context(), nodeID, vmid)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list scheduled actions")
	}
	if actions == nil {
		actions = []model.ScheduledAction{}
	}
	return apiPkg.Success(c, actions)
}

func (h *VMHealthHandler) CreateScheduledAction(c echo.Context) error {
	var req model.CreateScheduledActionRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	action, err := h.scheduledSvc.Create(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create scheduled action")
	}
	return apiPkg.Created(c, action)
}

func (h *VMHealthHandler) DeleteScheduledAction(c echo.Context) error {
	id, err := uuid.Parse(c.Param("actionId"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid action id")
	}

	if err := h.scheduledSvc.Delete(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "failed to delete scheduled action")
	}
	return apiPkg.NoContent(c)
}

// --- VM Dependencies ---

func (h *VMHealthHandler) ListAllDependencies(c echo.Context) error {
	deps, err := h.depSvc.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list dependencies")
	}
	if deps == nil {
		deps = []model.VMDependency{}
	}
	return apiPkg.Success(c, deps)
}

func (h *VMHealthHandler) ListVMDependencies(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	deps, err := h.depSvc.ListByVM(c.Request().Context(), nodeID, vmid)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list vm dependencies")
	}
	if deps == nil {
		deps = []model.VMDependency{}
	}
	return apiPkg.Success(c, deps)
}

func (h *VMHealthHandler) CreateDependency(c echo.Context) error {
	var req model.CreateVMDependencyRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	dep, err := h.depSvc.Create(c.Request().Context(), req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create dependency")
	}
	return apiPkg.Created(c, dep)
}

func (h *VMHealthHandler) DeleteDependency(c echo.Context) error {
	id, err := uuid.Parse(c.Param("depId"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid dependency id")
	}

	if err := h.depSvc.Delete(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "failed to delete dependency")
	}
	return apiPkg.NoContent(c)
}
