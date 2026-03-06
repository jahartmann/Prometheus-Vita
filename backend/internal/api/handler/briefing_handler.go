package handler

import (
	"strconv"

	"github.com/antigravity/prometheus/internal/model"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/service/briefing"
	"github.com/labstack/echo/v4"
)

type BriefingHandler struct {
	service *briefing.Service
}

func NewBriefingHandler(service *briefing.Service) *BriefingHandler {
	return &BriefingHandler{service: service}
}

func (h *BriefingHandler) GetLatest(c echo.Context) error {
	b, err := h.service.GetLatest(c.Request().Context())
	if err != nil {
		return apiPkg.NotFound(c, "Noch kein Morning Briefing verfuegbar")
	}
	return apiPkg.Success(c, b)
}

func (h *BriefingHandler) GetLiveSummary(c echo.Context) error {
	summary, err := h.service.GetLiveSummary(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Erstellen des Live-Briefings")
	}
	return apiPkg.Success(c, summary)
}

func (h *BriefingHandler) List(c echo.Context) error {
	limit := 10
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	briefings, err := h.service.List(c.Request().Context(), limit)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Briefings")
	}
	if briefings == nil {
		briefings = []model.MorningBriefing{}
	}
	return apiPkg.Success(c, briefings)
}
