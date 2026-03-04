package middleware

import (
	"github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/labstack/echo/v4"
)

func RequireRole(roles ...model.UserRole) echo.MiddlewareFunc {
	allowed := make(map[model.UserRole]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get(ContextKeyRole).(model.UserRole)
			if !ok {
				return api.Unauthorized(c, "no role found in context")
			}

			if !allowed[role] {
				return api.Forbidden(c, "insufficient permissions")
			}

			return next(c)
		}
	}
}
