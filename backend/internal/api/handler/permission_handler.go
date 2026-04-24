package handler

import (
	"context"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PermissionHandler struct {
	rolePermissionRepo repository.RolePermissionRepository
}

func NewPermissionHandler(rolePermissionRepo repository.RolePermissionRepository) *PermissionHandler {
	return &PermissionHandler{rolePermissionRepo: rolePermissionRepo}
}

func (h *PermissionHandler) GetCatalog(c echo.Context) error {
	return apiPkg.Success(c, h.catalog(c.Request().Context()))
}

func (h *PermissionHandler) UpdateRole(c echo.Context) error {
	role := model.UserRole(c.Param("role"))
	if !role.IsValid() {
		return apiPkg.BadRequest(c, "invalid role")
	}
	if h.rolePermissionRepo == nil {
		return apiPkg.InternalError(c, "role permission repository is not available")
	}

	var req model.UpdateRolePermissionsRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	permissions, err := model.NormalizeRolePermissions(role, req.Permissions)
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	userID, _ := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	var updatedBy *uuid.UUID
	if userID != uuid.Nil {
		updatedBy = &userID
	}

	updated, err := h.rolePermissionRepo.Update(c.Request().Context(), role, permissions, updatedBy)
	if err != nil {
		return apiPkg.InternalError(c, "failed to update role permissions")
	}

	return apiPkg.Success(c, model.RolePermissionSummary{
		Role:        updated.Role,
		Permissions: updated.Permissions,
		UpdatedAt:   &updated.UpdatedAt,
		UpdatedBy:   updated.UpdatedBy,
	})
}

func (h *PermissionHandler) catalog(ctx context.Context) model.PermissionCatalog {
	if h.rolePermissionRepo == nil {
		return model.BuildPermissionCatalog()
	}

	overrides, err := h.rolePermissionRepo.List(ctx)
	if err != nil {
		return model.BuildPermissionCatalog()
	}

	rolesByName := map[model.UserRole]model.RolePermissionSummary{}
	for _, override := range overrides {
		permissions, err := model.NormalizeRolePermissions(override.Role, override.Permissions)
		if err != nil {
			continue
		}
		updatedAt := override.UpdatedAt
		rolesByName[override.Role] = model.RolePermissionSummary{
			Role:        override.Role,
			Permissions: permissions,
			UpdatedAt:   &updatedAt,
			UpdatedBy:   override.UpdatedBy,
		}
	}

	roles := make([]model.RolePermissionSummary, 0, 3)
	for _, role := range []model.UserRole{model.RoleAdmin, model.RoleOperator, model.RoleViewer} {
		if summary, ok := rolesByName[role]; ok {
			roles = append(roles, summary)
			continue
		}
		roles = append(roles, model.RolePermissionSummary{Role: role, Permissions: model.RolePermissions(role)})
	}

	return model.BuildPermissionCatalogWithRoles(roles)
}
