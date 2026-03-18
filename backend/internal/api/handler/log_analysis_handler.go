package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/loganalyzer"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type LogAnalysisHandler struct {
	reporter     *loganalyzer.Reporter
	anomalyRepo  repository.LogAnomalyRepository
	analysisRepo repository.LogAnalysisRepository
}

func NewLogAnalysisHandler(
	reporter *loganalyzer.Reporter,
	anomalyRepo repository.LogAnomalyRepository,
	analysisRepo repository.LogAnalysisRepository,
) *LogAnalysisHandler {
	return &LogAnalysisHandler{
		reporter:     reporter,
		anomalyRepo:  anomalyRepo,
		analysisRepo: analysisRepo,
	}
}

func (h *LogAnalysisHandler) Analyze(c echo.Context) error {
	var req model.AnalyzeLogsRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	analysis, err := h.reporter.Analyze(c.Request().Context(), req)
	if err != nil {
		return response.InternalError(c, "failed to analyze logs")
	}
	return response.Success(c, analysis)
}

func (h *LogAnalysisHandler) ListAnomalies(c echo.Context) error {
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

	anomalies, err := h.anomalyRepo.ListByNode(c.Request().Context(), nodeID, limit, offset)
	if err != nil {
		return response.InternalError(c, "failed to list anomalies")
	}
	if anomalies == nil {
		anomalies = []model.LogAnomaly{}
	}
	return response.Success(c, anomalies)
}

func (h *LogAnalysisHandler) GetAnomaly(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}
	anomaly, err := h.anomalyRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "anomaly not found")
		}
		return response.InternalError(c, "failed to get anomaly")
	}
	return response.Success(c, anomaly)
}

func (h *LogAnalysisHandler) Acknowledge(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid anomaly id")
	}
	userID := c.Get("user_id").(uuid.UUID)
	if err := h.anomalyRepo.Acknowledge(c.Request().Context(), id, userID); err != nil {
		return response.InternalError(c, "failed to acknowledge anomaly")
	}
	return response.NoContent(c)
}

func (h *LogAnalysisHandler) ListAnalyses(c echo.Context) error {
	var nodeIDs []uuid.UUID
	if raw := c.QueryParam("node_ids"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := uuid.Parse(part)
			if err != nil {
				return response.BadRequest(c, "invalid node_ids")
			}
			nodeIDs = append(nodeIDs, id)
		}
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

	analyses, err := h.analysisRepo.ListByNodes(c.Request().Context(), nodeIDs, limit, offset)
	if err != nil {
		return response.InternalError(c, "failed to list analyses")
	}
	if analyses == nil {
		analyses = []model.LogAnalysis{}
	}
	return response.Success(c, analyses)
}
