package handler

import (
	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type LogBookmarkHandler struct {
	repo repository.LogBookmarkRepository
}

func NewLogBookmarkHandler(repo repository.LogBookmarkRepository) *LogBookmarkHandler {
	return &LogBookmarkHandler{repo: repo}
}

func (h *LogBookmarkHandler) Create(c echo.Context) error {
	var req model.CreateLogBookmarkRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	bookmark := &model.LogBookmark{
		NodeID:       req.NodeID,
		AnomalyID:    req.AnomalyID,
		LogEntryJSON: req.LogEntryJSON,
		UserNote:     req.UserNote,
	}
	if err := h.repo.Create(c.Request().Context(), bookmark); err != nil {
		return response.InternalError(c, "failed to create bookmark")
	}
	return response.Created(c, bookmark)
}

func (h *LogBookmarkHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}
	bookmarks, err := h.repo.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return response.InternalError(c, "failed to list bookmarks")
	}
	if bookmarks == nil {
		bookmarks = []model.LogBookmark{}
	}
	return response.Success(c, bookmarks)
}

func (h *LogBookmarkHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}
	if err := h.repo.Delete(c.Request().Context(), id); err != nil {
		return response.InternalError(c, "failed to delete bookmark")
	}
	return response.NoContent(c)
}
