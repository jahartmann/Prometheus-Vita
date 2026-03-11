package handler

import (
	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	vmService "github.com/antigravity/prometheus/internal/service/vm"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type VMGroupHandler struct {
	groupSvc *vmService.GroupService
}

func NewVMGroupHandler(groupSvc *vmService.GroupService) *VMGroupHandler {
	return &VMGroupHandler{groupSvc: groupSvc}
}

func (h *VMGroupHandler) List(c echo.Context) error {
	groups, err := h.groupSvc.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list vm groups")
	}
	if groups == nil {
		groups = []model.VMGroup{}
	}
	return apiPkg.Success(c, groups)
}

func (h *VMGroupHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid group id")
	}
	group, err := h.groupSvc.GetByID(c.Request().Context(), id)
	if err != nil {
		return apiPkg.NotFound(c, "vm group not found")
	}
	return apiPkg.Success(c, group)
}

func (h *VMGroupHandler) Create(c echo.Context) error {
	var req model.CreateVMGroupRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Name == "" {
		return apiPkg.BadRequest(c, "name is required")
	}

	createdBy, _ := c.Get(middleware.ContextKeyUserID).(uuid.UUID)

	group, err := h.groupSvc.Create(c.Request().Context(), &req, createdBy)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create vm group")
	}
	return apiPkg.Created(c, group)
}

func (h *VMGroupHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid group id")
	}

	var req model.UpdateVMGroupRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	group, err := h.groupSvc.Update(c.Request().Context(), id, &req)
	if err != nil {
		return apiPkg.InternalError(c, "failed to update vm group")
	}
	return apiPkg.Success(c, group)
}

func (h *VMGroupHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid group id")
	}
	if err := h.groupSvc.Delete(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "failed to delete vm group")
	}
	return apiPkg.NoContent(c)
}

func (h *VMGroupHandler) ListMembers(c echo.Context) error {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid group id")
	}
	members, err := h.groupSvc.ListMembers(c.Request().Context(), groupID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list group members")
	}
	if members == nil {
		members = []model.VMGroupMember{}
	}
	return apiPkg.Success(c, members)
}

func (h *VMGroupHandler) AddMember(c echo.Context) error {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid group id")
	}

	var req model.AddVMGroupMemberRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	nodeID, err := uuid.Parse(req.NodeID)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node_id")
	}

	if err := h.groupSvc.AddMember(c.Request().Context(), groupID, nodeID, req.VMID); err != nil {
		return apiPkg.InternalError(c, "failed to add member to group")
	}

	return apiPkg.Success(c, map[string]string{"status": "ok"})
}

func (h *VMGroupHandler) RemoveMember(c echo.Context) error {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid group id")
	}

	var req model.RemoveVMGroupMemberRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	nodeID, err := uuid.Parse(req.NodeID)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node_id")
	}

	if err := h.groupSvc.RemoveMember(c.Request().Context(), groupID, nodeID, req.VMID); err != nil {
		return apiPkg.InternalError(c, "failed to remove member from group")
	}

	return apiPkg.NoContent(c)
}
