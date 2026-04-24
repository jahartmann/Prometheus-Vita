package handler

import (
	"fmt"
	"strconv"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	vmService "github.com/antigravity/prometheus/internal/service/vm"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type VMPermissionHandler struct {
	permSvc *vmService.PermissionService
}

func NewVMPermissionHandler(permSvc *vmService.PermissionService) *VMPermissionHandler {
	return &VMPermissionHandler{permSvc: permSvc}
}

func (h *VMPermissionHandler) List(c echo.Context) error {
	userIDStr := c.QueryParam("user_id")
	if userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return apiPkg.BadRequest(c, "invalid user_id")
		}
		perms, err := h.permSvc.ListByUser(c.Request().Context(), userID)
		if err != nil {
			return apiPkg.InternalError(c, "failed to list permissions")
		}
		if perms == nil {
			perms = []model.VMPermission{}
		}
		return apiPkg.Success(c, perms)
	}

	targetType := c.QueryParam("target_type")
	targetID := c.QueryParam("target_id")
	nodeIDStr := c.QueryParam("node_id")
	if targetType != "" || targetID != "" || nodeIDStr != "" {
		if targetType == "" || targetID == "" || nodeIDStr == "" {
			return apiPkg.BadRequest(c, "target_type, target_id, and node_id are required when filtering by target")
		}
		if !model.IsValidVMPermissionTarget(targetType) {
			return apiPkg.BadRequest(c, "target_type must be 'vm', 'group', 'node', or 'environment'")
		}
	}

	if targetType != "" && targetID != "" && nodeIDStr != "" {
		nodeID, err := uuid.Parse(nodeIDStr)
		if err != nil {
			return apiPkg.BadRequest(c, "invalid node_id")
		}
		perms, err := h.permSvc.ListByTarget(c.Request().Context(), targetType, targetID, nodeID)
		if err != nil {
			return apiPkg.InternalError(c, "failed to list permissions")
		}
		if perms == nil {
			perms = []model.VMPermission{}
		}
		return apiPkg.Success(c, perms)
	}

	// No filters: list all permissions (admin use case for permission matrix)
	perms, err := h.permSvc.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list permissions")
	}
	if perms == nil {
		perms = []model.VMPermission{}
	}
	return apiPkg.Success(c, perms)
}

func (h *VMPermissionHandler) Create(c echo.Context) error {
	var perm model.VMPermission
	if err := c.Bind(&perm); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if perm.UserID == uuid.Nil || perm.TargetType == "" || perm.TargetID == "" || perm.NodeID == uuid.Nil {
		return apiPkg.BadRequest(c, "user_id, target_type, target_id, and node_id are required")
	}
	if !model.IsValidVMPermissionTarget(perm.TargetType) {
		return apiPkg.BadRequest(c, "target_type must be 'vm', 'group', 'node', or 'environment'")
	}
	if err := validateVMPermissionPayload(perm.TargetType, perm.TargetID, perm.NodeID, perm.Permissions); err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	if createdBy, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID); ok {
		perm.CreatedBy = createdBy
	}

	if err := h.permSvc.Grant(c.Request().Context(), &perm); err != nil {
		return apiPkg.InternalError(c, "failed to create permission")
	}

	return apiPkg.Created(c, perm)
}

func (h *VMPermissionHandler) Upsert(c echo.Context) error {
	var req struct {
		UserID      string   `json:"user_id"`
		TargetType  string   `json:"target_type"`
		TargetID    string   `json:"target_id"`
		NodeID      string   `json:"node_id"`
		Permissions []string `json:"permissions"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user_id")
	}

	nodeID, err := uuid.Parse(req.NodeID)
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node_id")
	}

	if !model.IsValidVMPermissionTarget(req.TargetType) {
		return apiPkg.BadRequest(c, "target_type must be 'vm', 'group', 'node', or 'environment'")
	}

	if req.TargetID == "" {
		return apiPkg.BadRequest(c, "target_id is required")
	}

	if err := validateVMPermissionPayload(req.TargetType, req.TargetID, nodeID, req.Permissions); err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	createdBy, _ := c.Get(middleware.ContextKeyUserID).(uuid.UUID)

	perm := &model.VMPermission{
		UserID:      userID,
		TargetType:  req.TargetType,
		TargetID:    req.TargetID,
		NodeID:      nodeID,
		Permissions: req.Permissions,
		CreatedBy:   createdBy,
	}

	if err := h.permSvc.Upsert(c.Request().Context(), perm); err != nil {
		return apiPkg.InternalError(c, "failed to upsert vm permission")
	}

	return apiPkg.Success(c, perm)
}

func (h *VMPermissionHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid permission id")
	}

	existing, err := h.permSvc.GetByID(c.Request().Context(), id)
	if err != nil {
		return apiPkg.NotFound(c, "permission not found")
	}

	var req struct {
		Permissions []string `json:"permissions"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if err := validateVMPermissionPayload(existing.TargetType, existing.TargetID, existing.NodeID, req.Permissions); err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	existing.Permissions = req.Permissions
	if err := h.permSvc.Update(c.Request().Context(), existing); err != nil {
		return apiPkg.InternalError(c, "failed to update permission")
	}

	return apiPkg.Success(c, existing)
}

func (h *VMPermissionHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid permission id")
	}

	if err := h.permSvc.Revoke(c.Request().Context(), id); err != nil {
		return apiPkg.InternalError(c, "failed to delete permission")
	}

	return apiPkg.NoContent(c)
}

func (h *VMPermissionHandler) GetEffective(c echo.Context) error {
	userID, err := uuid.Parse(c.QueryParam("user_id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid user_id")
	}
	nodeID, err := uuid.Parse(c.QueryParam("node_id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node_id")
	}
	vmid, err := strconv.Atoi(c.QueryParam("vmid"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid vmid")
	}

	perms, err := h.permSvc.GetEffectivePermissions(c.Request().Context(), userID, nodeID, vmid)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get effective permissions")
	}
	if perms == nil {
		perms = []string{}
	}
	return apiPkg.Success(c, perms)
}

func (h *VMPermissionHandler) ListAllPermissions(c echo.Context) error {
	return apiPkg.Success(c, model.AllVMPermissions)
}

func validateVMPermissionPayload(targetType, targetID string, nodeID uuid.UUID, permissions []string) error {
	if len(permissions) == 0 {
		return fmt.Errorf("permissions is required")
	}
	if targetID == "" {
		return fmt.Errorf("target_id is required")
	}
	if nodeID == uuid.Nil {
		return fmt.Errorf("node_id is required")
	}
	if targetType == model.VMPermissionTargetNode && targetID != nodeID.String() {
		return fmt.Errorf("node target_id must match node_id")
	}
	for _, permission := range permissions {
		if !model.IsValidVMPermission(permission) {
			return fmt.Errorf("invalid permission: %s", permission)
		}
	}
	return nil
}
