package middleware

import (
	"encoding/json"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/service/gateway"
	"github.com/labstack/echo/v4"
)

const ContextKeyAPITokenID = "api_token_id"
const ContextKeyAPIPermissions = "api_permissions"

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
				// API key was provided but is invalid — fail closed
				return response.Unauthorized(c, "invalid api key")
			}

			// Get user for this token
			user, err := gatewaySvc.GetUserForToken(c.Request().Context(), token)
			if err != nil {
				return response.Unauthorized(c, "failed to resolve api key user")
			}
			if !user.IsActive {
				return response.Forbidden(c, "api key user is inactive")
			}

			// Set context like JWT would
			c.Set(ContextKeyUserID, user.ID)
			c.Set(ContextKeyUsername, user.Username)
			c.Set(ContextKeyRole, user.Role)
			c.Set(ContextKeyAPITokenID, token.ID)
			c.Set(ContextKeyAPIPermissions, parseAPITokenPermissions(token.Permissions))

			return next(c)
		}
	}
}

func parseAPITokenPermissions(raw json.RawMessage) []model.Permission {
	var values []string
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	permissions := make([]model.Permission, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		permissions = append(permissions, model.Permission(value))
	}
	return permissions
}
