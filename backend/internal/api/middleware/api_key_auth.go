package middleware

import (
	"github.com/antigravity/prometheus/internal/service/gateway"
	"github.com/labstack/echo/v4"
)

const ContextKeyAPITokenID = "api_token_id"

func APIKeyAuth(gatewaySvc *gateway.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey == "" {
				// No API key - let other auth middleware handle it
				return next(c)
			}

			token, err := gatewaySvc.ValidateToken(c.Request().Context(), apiKey)
			if err != nil {
				// Invalid key, let JWT middleware handle auth
				return next(c)
			}

			// Get user for this token
			user, err := gatewaySvc.GetUserForToken(c.Request().Context(), token)
			if err != nil {
				return next(c)
			}

			// Set context like JWT would
			c.Set(ContextKeyUserID, user.ID)
			c.Set(ContextKeyUsername, user.Username)
			c.Set(ContextKeyRole, user.Role)
			c.Set(ContextKeyAPITokenID, token.ID)

			return next(c)
		}
	}
}
