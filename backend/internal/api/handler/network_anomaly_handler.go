package handler

import (
	"strconv"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type NetworkAnomalyHandler struct {
	repo repository.NetworkAnomalyRepository
}

func NewNetworkAnomalyHandler(repo repository.NetworkAnomalyRepository) *NetworkAnomalyHandler {
	return &NetworkAnomalyHandler{repo: repo}
}

func (h *NetworkAnomalyHandler) ListByNode(c echo.Context) error {
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

	anomalies, err := h.repo.ListByNode(c.Request().Context(), nodeID, limit, offset)
	if err != nil {
		return response.InternalError(c, "failed to list network anomalies")
	}
	if anomalies == nil {
		anomalies = []model.NetworkAnomaly{}
	}
	return response.Success(c, anomalies)
}

func (h *NetworkAnomalyHandler) Acknowledge(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}
	userID := c.Get("user_id").(uuid.UUID)
	if err := h.repo.Acknowledge(c.Request().Context(), id, userID); err != nil {
		return response.InternalError(c, "failed to acknowledge network anomaly")
	}
	return response.NoContent(c)
}
