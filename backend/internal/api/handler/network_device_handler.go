package handler

import (
	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type NetworkDeviceHandler struct {
	repo repository.NetworkDeviceRepository
}

func NewNetworkDeviceHandler(repo repository.NetworkDeviceRepository) *NetworkDeviceHandler {
	return &NetworkDeviceHandler{repo: repo}
}

func (h *NetworkDeviceHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}
	devices, err := h.repo.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return response.InternalError(c, "failed to list network devices")
	}
	if devices == nil {
		devices = []model.NetworkDevice{}
	}
	return response.Success(c, devices)
}

func (h *NetworkDeviceHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid device id")
	}

	var req model.UpdateNetworkDeviceRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if err := h.repo.Update(c.Request().Context(), id, req); err != nil {
		return response.InternalError(c, "failed to update network device")
	}
	return response.Success(c, map[string]string{"status": "updated"})
}
