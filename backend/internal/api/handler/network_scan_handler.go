package handler

import (
	"errors"
	"strconv"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/netscan"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type NetworkScanHandler struct {
	scheduler *netscan.ScanScheduler
	scanRepo  repository.NetworkScanRepository
}

func NewNetworkScanHandler(scheduler *netscan.ScanScheduler, scanRepo repository.NetworkScanRepository) *NetworkScanHandler {
	return &NetworkScanHandler{
		scheduler: scheduler,
		scanRepo:  scanRepo,
	}
}

func (h *NetworkScanHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}

	limit := 50
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}

	scans, err := h.scanRepo.ListByNode(c.Request().Context(), nodeID, limit, offset)
	if err != nil {
		return response.InternalError(c, "failed to list network scans")
	}
	if scans == nil {
		scans = []model.NetworkScan{}
	}
	return response.Success(c, scans)
}

func (h *NetworkScanHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}
	scan, err := h.scanRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "scan not found")
		}
		return response.InternalError(c, "failed to get network scan")
	}
	return response.Success(c, scan)
}

func (h *NetworkScanHandler) Trigger(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}

	var req model.TriggerScanRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	scan, err := h.scheduler.TriggerScan(c.Request().Context(), nodeID, req.ScanType)
	if err != nil {
		return response.InternalError(c, "failed to trigger network scan")
	}
	return response.Created(c, scan)
}

func (h *NetworkScanHandler) Diff(c echo.Context) error {
	id1, err := uuid.Parse(c.Param("id1"))
	if err != nil {
		return response.BadRequest(c, "invalid id1")
	}
	id2, err := uuid.Parse(c.Param("id2"))
	if err != nil {
		return response.BadRequest(c, "invalid id2")
	}

	diff, err := h.scheduler.GetDiff(c.Request().Context(), id1, id2)
	if err != nil {
		return response.InternalError(c, "failed to compute scan diff")
	}
	return response.Success(c, diff)
}
