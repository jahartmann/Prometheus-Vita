package middleware

import (
	"strings"

	"github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/service/auth"
	"github.com/labstack/echo/v4"
)

const (
	ContextKeyUserID   = "user_id"
	ContextKeyUsername = "username"
	ContextKeyRole     = "role"
)

func JWTAuth(jwtSvc *auth.JWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip if already authenticated via API key
			if c.Get(ContextKeyUserID) != nil {
				return next(c)
			}

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return api.Unauthorized(c, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return api.Unauthorized(c, "invalid authorization header format")
			}

			claims, err := jwtSvc.ValidateAccessToken(parts[1])
			if err != nil {
				return api.Unauthorized(c, "invalid or expired token")
			}

			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyUsername, claims.Username)
			c.Set(ContextKeyRole, claims.Role)

			return next(c)
		}
	}
}
