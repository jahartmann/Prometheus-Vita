package middleware

import (
	"errors"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/labstack/echo/v4"
)

const ContextKeyRolePermissions = "role_permissions"

func RequireRole(roles ...model.UserRole) echo.MiddlewareFunc {
	allowed := make(map[model.UserRole]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get(ContextKeyRole).(model.UserRole)
			if !ok {
				return response.Unauthorized(c, "no role found in context")
			}

			if !allowed[role] {
				return response.Forbidden(c, "insufficient permissions")
			}

			return next(c)
		}
	}
}

func LoadRolePermissions(repo repository.RolePermissionRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get(ContextKeyRole).(model.UserRole)
			if !ok {
				return next(c)
			}

			permissions, err := repo.Get(c.Request().Context(), role)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					c.Set(ContextKeyRolePermissions, model.RolePermissions(role))
					return next(c)
				}
				return response.InternalError(c, "failed to load role permissions")
			}

			c.Set(ContextKeyRolePermissions, permissions.Permissions)
			return next(c)
		}
	}
}

func RequirePermission(permission model.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get(ContextKeyRole).(model.UserRole)
			if !ok {
				return response.Unauthorized(c, "no role found in context")
			}

			rolePermissions := model.RolePermissions(role)
			if loadedPermissions, ok := c.Get(ContextKeyRolePermissions).([]model.Permission); ok {
				rolePermissions = loadedPermissions
			}

			if !model.NewPermissionSet(rolePermissions).Allows(permission) {
				return response.Forbidden(c, "insufficient permissions")
			}

			if tokenPermissions, ok := c.Get(ContextKeyAPIPermissions).([]model.Permission); ok {
				if !model.NewPermissionSet(tokenPermissions).Allows(permission) {
					return response.Forbidden(c, "api token scope does not allow this action")
				}
			}

			return next(c)
		}
	}
}
