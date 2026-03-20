package middleware

import (
	"strings"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MustChangePassword blocks access to all protected routes (except password
// change endpoints) when the authenticated user has must_change_password set.
func MustChangePassword(userRepo repository.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Allow password-change and auth endpoints through
			path := c.Request().URL.Path
			if strings.HasSuffix(path, "/password") ||
				strings.HasPrefix(path, "/api/v1/auth/") {
				return next(c)
			}

			userID, ok := c.Get(ContextKeyUserID).(uuid.UUID)
			if !ok {
				return next(c)
			}

			user, err := userRepo.GetByID(c.Request().Context(), userID)
			if err != nil {
				return next(c)
			}

			if user.MustChangePassword {
				return response.ErrorResponse(c, 403, "Passwort muss zuerst geaendert werden")
			}

			return next(c)
		}
	}
}
