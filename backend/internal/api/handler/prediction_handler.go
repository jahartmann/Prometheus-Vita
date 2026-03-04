package handler

import (
	"github.com/antigravity/prometheus/internal/model"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/service/prediction"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PredictionHandler struct {
	service *prediction.Service
}

func NewPredictionHandler(service *prediction.Service) *PredictionHandler {
	return &PredictionHandler{service: service}
}

func (h *PredictionHandler) ListCritical(c echo.Context) error {
	preds, err := h.service.ListCritical(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Vorhersagen")
	}
	if preds == nil {
		preds = []model.MaintenancePrediction{}
	}
	return apiPkg.Success(c, preds)
}

func (h *PredictionHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Node-ID")
	}

	preds, err := h.service.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Vorhersagen")
	}
	if preds == nil {
		preds = []model.MaintenancePrediction{}
	}
	return apiPkg.Success(c, preds)
}
