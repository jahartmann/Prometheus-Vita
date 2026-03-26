package handler

import (
	"strconv"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
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

// VM Tag handlers

func (h *TagHandler) GetVMTags(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}
	if vmid <= 0 {
		return apiPkg.BadRequest(c, "vmid must be positive")
	}

	tags, err := h.tagRepo.GetByVM(c.Request().Context(), nodeID, vmid)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get vm tags")
	}

	if tags == nil {
		tags = []model.Tag{}
	}

	return apiPkg.Success(c, tags)
}

func (h *TagHandler) AddTagToVM(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}
	if vmid <= 0 {
		return apiPkg.BadRequest(c, "vmid must be positive")
	}

	var req struct {
		TagID  string `json:"tag_id"`
		VMType string `json:"vm_type"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	tagID, err := uuid.Parse(req.TagID)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid tag_id")
	}

	vmType := req.VMType
	if vmType == "" {
		vmType = "qemu"
	}

	if err := h.tagRepo.AddToVM(c.Request().Context(), nodeID, vmid, vmType, tagID); err != nil {
		return apiPkg.InternalError(c, "failed to add tag to vm")
	}

	return apiPkg.Success(c, map[string]string{"status": "ok"})
}

func (h *TagHandler) RemoveTagFromVM(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}
	if vmid <= 0 {
		return apiPkg.BadRequest(c, "vmid must be positive")
	}

	tagID, err := uuid.Parse(c.Param("tagId"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid tag id")
	}

	if err := h.tagRepo.RemoveFromVM(c.Request().Context(), nodeID, vmid, tagID); err != nil {
		return apiPkg.InternalError(c, "failed to remove tag from vm")
	}

	return apiPkg.NoContent(c)
}

func (h *TagHandler) BulkAssignTag(c echo.Context) error {
	tagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid tag id")
	}

	var req model.BulkTagRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if len(req.Targets) == 0 {
		return apiPkg.BadRequest(c, "no targets provided")
	}

	vmTags := make([]model.VMTag, 0, len(req.Targets))
	for _, t := range req.Targets {
		nid, err := uuid.Parse(t.NodeID)
		if err != nil {
			return apiPkg.BadRequest(c, "invalid node_id in targets: "+t.NodeID)
		}
		vmType := t.VMType
		if vmType == "" {
			vmType = "qemu"
		}
		vmTags = append(vmTags, model.VMTag{
			NodeID: nid,
			VMID:   t.VMID,
			VMType: vmType,
			TagID:  tagID,
		})
	}

	if err := h.tagRepo.BulkAddToVMs(c.Request().Context(), vmTags); err != nil {
		return apiPkg.InternalError(c, "failed to bulk assign tag")
	}

	return apiPkg.Success(c, map[string]interface{}{
		"status":   "ok",
		"assigned": len(vmTags),
	})
}

func (h *TagHandler) BulkRemoveTag(c echo.Context) error {
	tagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid tag id")
	}

	var req model.BulkTagRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if len(req.Targets) == 0 {
		return apiPkg.BadRequest(c, "no targets provided")
	}

	// Group by node_id for efficient batch removal
	nodeVMIDs := make(map[uuid.UUID][]int)
	for _, t := range req.Targets {
		nid, err := uuid.Parse(t.NodeID)
		if err != nil {
			return apiPkg.BadRequest(c, "invalid node_id in targets: "+t.NodeID)
		}
		nodeVMIDs[nid] = append(nodeVMIDs[nid], t.VMID)
	}

	removed := 0
	for nid, vmids := range nodeVMIDs {
		if err := h.tagRepo.BulkRemoveTagFromVMs(c.Request().Context(), tagID, nid, vmids); err != nil {
			return apiPkg.InternalError(c, "failed to bulk remove tag")
		}
		removed += len(vmids)
	}

	return apiPkg.Success(c, map[string]interface{}{
		"status":  "ok",
		"removed": removed,
	})
}

func (h *TagHandler) GetVMsByTag(c echo.Context) error {
	tagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid tag id")
	}

	vmTags, err := h.tagRepo.GetVMsByTag(c.Request().Context(), tagID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get vms by tag")
	}

	if vmTags == nil {
		vmTags = []model.VMTag{}
	}

	return apiPkg.Success(c, vmTags)
}
