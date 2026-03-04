package handler

import (
	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/service/topology"
	"github.com/labstack/echo/v4"
)

type TopologyHandler struct {
	service *topology.Service
}

func NewTopologyHandler(service *topology.Service) *TopologyHandler {
	return &TopologyHandler{service: service}
}

func (h *TopologyHandler) GetTopology(c echo.Context) error {
	graph, err := h.service.BuildTopology(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "Topologie konnte nicht erstellt werden")
	}
	return apiPkg.Success(c, graph)
}
