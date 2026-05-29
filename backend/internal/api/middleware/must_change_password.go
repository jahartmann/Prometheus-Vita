package middleware

import (
	"net/http"
	"strings"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MustChangePassword blocks access for inactive users and for users who must
// change their password before using protected routes.
func MustChangePassword(userRepo repository.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			method := c.Request().Method

			// Endpoints required to complete the forced password change:
			// logout/refresh, the current-user lookup, and the password policy
			// the change-password page renders as requirements.
			if path == "/api/v1/auth/logout" ||
				path == "/api/v1/auth/refresh" ||
				path == "/api/v1/auth/me" ||
				(method == http.MethodGet && path == "/api/v1/password-policy") {
				return next(c)
			}

			// The password-change route itself: POST /api/v1/users/<id>/password.
			// The route is registered as /users/:id/password, so match by
			// prefix+suffix rather than a brittle literal that never matched the
			// real (uuid) path — which previously locked the seeded admin out.
			if method == http.MethodPost &&
				strings.HasPrefix(path, "/api/v1/users/") &&
				strings.HasSuffix(path, "/password") {
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

			if !user.IsActive {
				return response.ErrorResponse(c, 403, "Konto ist deaktiviert")
			}

			if user.MustChangePassword {
				return response.ErrorResponse(c, 403, "Passwort muss zuerst geaendert werden")
			}

			return next(c)
		}
	}
}
