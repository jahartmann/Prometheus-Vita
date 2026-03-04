package handler

import (
	"github.com/antigravity/prometheus/internal/model"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/service/anomaly"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AnomalyHandler struct {
	service *anomaly.Service
}

func NewAnomalyHandler(service *anomaly.Service) *AnomalyHandler {
	return &AnomalyHandler{service: service}
}

func (h *AnomalyHandler) ListUnresolved(c echo.Context) error {
	records, err := h.service.ListUnresolved(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Anomalien")
	}
	if records == nil {
		records = []model.AnomalyRecord{}
	}
	return apiPkg.Success(c, records)
}

func (h *AnomalyHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Node-ID")
	}

	records, err := h.service.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Anomalien")
	}
	if records == nil {
		records = []model.AnomalyRecord{}
	}
	return apiPkg.Success(c, records)
}

func (h *AnomalyHandler) Resolve(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Anomalie-ID")
	}

	if err := h.service.Resolve(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "Fehler beim Loesen der Anomalie")
	}
	return apiPkg.NoContent(c)
}
