package handler

import (
	"strconv"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/intelligence"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type SecurityHandler struct {
	repo        repository.SecurityEventRepository
	analysisSvc *intelligence.Service
}

func NewSecurityHandler(repo repository.SecurityEventRepository, analysisSvc *intelligence.Service) *SecurityHandler {
	return &SecurityHandler{repo: repo, analysisSvc: analysisSvc}
}

func (h *SecurityHandler) ListUnacknowledged(c echo.Context) error {
	events, err := h.repo.ListUnacknowledged(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Security-Events")
	}
	if events == nil {
		events = []model.SecurityEvent{}
	}
	return apiPkg.Success(c, events)
}

func (h *SecurityHandler) ListRecent(c echo.Context) error {
	limit := 50
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	events, err := h.repo.ListRecent(c.Request().Context(), limit)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Security-Events")
	}
	if events == nil {
		events = []model.SecurityEvent{}
	}
	return apiPkg.Success(c, events)
}

func (h *SecurityHandler) GetStats(c echo.Context) error {
	ctx := c.Request().Context()

	bySeverity, err := h.repo.CountBySeverity(ctx)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Statistiken")
	}

	byCategory, err := h.repo.CountByCategory(ctx)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Statistiken")
	}

	total := 0
	for _, v := range bySeverity {
		total += v
	}

	stats := model.SecurityEventStats{
		Total:          total,
		Unacknowledged: total,
		BySeverity:     bySeverity,
		ByCategory: byCategory,
	}

	return apiPkg.Success(c, stats)
}

func (h *SecurityHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Node-ID")
	}

	events, err := h.repo.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return apiPkg.InternalError(c, "Fehler beim Abrufen der Security-Events")
	}
	if events == nil {
		events = []model.SecurityEvent{}
	}
	return apiPkg.Success(c, events)
}

func (h *SecurityHandler) Acknowledge(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "Ungueltige Event-ID")
	}

	if err := h.repo.Acknowledge(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "Fehler beim Bestaetigen des Events")
	}
	return apiPkg.NoContent(c)
}

// GetMode returns the current analysis mode.
func (h *SecurityHandler) GetMode(c echo.Context) error {
	mode := string(h.analysisSvc.GetMode())
	return apiPkg.Success(c, map[string]string{"mode": mode})
}

// SetMode changes the analysis mode (hybrid, full_llm, rule_only).
func (h *SecurityHandler) SetMode(c echo.Context) error {
	var req struct {
		Mode string `json:"mode"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "Ungueltiger Request")
	}

	switch intelligence.AnalysisMode(req.Mode) {
	case intelligence.ModeHybrid, intelligence.ModeFullLLM, intelligence.ModeRuleOnly:
		h.analysisSvc.SetMode(intelligence.AnalysisMode(req.Mode))
		return apiPkg.Success(c, map[string]string{"mode": req.Mode})
	default:
		return apiPkg.BadRequest(c, "Ungueltiger Modus. Erlaubt: hybrid, full_llm, rule_only")
	}
}
