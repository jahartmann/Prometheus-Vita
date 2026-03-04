package handler

import (
	apiPkg "github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type TagHandler struct {
	tagRepo repository.TagRepository
}

func NewTagHandler(tagRepo repository.TagRepository) *TagHandler {
	return &TagHandler{tagRepo: tagRepo}
}

func (h *TagHandler) ListTags(c echo.Context) error {
	tags, err := h.tagRepo.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list tags")
	}

	if tags == nil {
		tags = []model.Tag{}
	}

	return apiPkg.Success(c, tags)
}

func (h *TagHandler) CreateTag(c echo.Context) error {
	var req model.CreateTagRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" {
		return apiPkg.BadRequest(c, "name is required")
	}
	if req.Color == "" {
		req.Color = "#6366f1"
	}

	tag := &model.Tag{
		Name:     req.Name,
		Color:    req.Color,
		Category: req.Category,
	}

	if err := h.tagRepo.Create(c.Request().Context(), tag); err != nil {
		return apiPkg.InternalError(c, "failed to create tag")
	}

	return apiPkg.Created(c, tag)
}

func (h *TagHandler) DeleteTag(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid tag id")
	}

	if err := h.tagRepo.Delete(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "failed to delete tag")
	}

	return apiPkg.NoContent(c)
}

func (h *TagHandler) AddTagToNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	var req model.AssignTagRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	tagID, err := uuid.Parse(req.TagID)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid tag_id")
	}

	if err := h.tagRepo.AddToNode(c.Request().Context(), nodeID, tagID); err != nil {
		return apiPkg.InternalError(c, "failed to add tag to node")
	}

	return apiPkg.Success(c, map[string]string{"status": "ok"})
}

func (h *TagHandler) RemoveTagFromNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	tagID, err := uuid.Parse(c.Param("tagId"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid tag id")
	}

	if err := h.tagRepo.RemoveFromNode(c.Request().Context(), nodeID, tagID); err != nil {
		return apiPkg.InternalError(c, "failed to remove tag from node")
	}

	return apiPkg.NoContent(c)
}

func (h *TagHandler) GetNodeTags(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	tags, err := h.tagRepo.GetByNode(c.Request().Context(), nodeID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get node tags")
	}

	if tags == nil {
		tags = []model.Tag{}
	}

	return apiPkg.Success(c, tags)
}
