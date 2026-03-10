package handler

import (
	"errors"
	"log/slog"
	"net/http"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/auth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authService *auth.Service
	userRepo    repository.UserRepository
}

func NewAuthHandler(authService *auth.Service, userRepo repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userRepo:    userRepo,
	}
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req model.LoginRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.Username == "" || req.Password == "" {
		return apiPkg.BadRequest(c, "Benutzername und Passwort sind erforderlich")
	}

	resp, err := h.authService.Login(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, auth.ErrAccountLocked) {
			return apiPkg.ErrorResponse(c, http.StatusTooManyRequests, "Konto voruebergehend gesperrt. Bitte in 15 Minuten erneut versuchen.")
		}
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return apiPkg.Unauthorized(c, "Benutzername oder Passwort ist falsch")
		}
		if errors.Is(err, auth.ErrUserInactive) {
			return apiPkg.Forbidden(c, "Dieses Konto ist deaktiviert")
		}
		slog.Error("login failed", slog.Any("error", err))
		return apiPkg.InternalError(c, "Anmeldung fehlgeschlagen – interner Serverfehler. Bitte Backend-Logs pruefen.")
	}

	return apiPkg.Success(c, resp)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if req.RefreshToken != "" {
		_ = h.authService.Logout(c.Request().Context(), req.RefreshToken)
	}

	return apiPkg.Success(c, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	var req model.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if req.RefreshToken == "" {
		return apiPkg.BadRequest(c, "refresh_token is required")
	}

	resp, err := h.authService.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) ||
			errors.Is(err, auth.ErrTokenRevoked) ||
			errors.Is(err, auth.ErrTokenExpired) {
			return apiPkg.Unauthorized(c, "invalid or expired refresh token")
		}
		return apiPkg.InternalError(c, "token refresh failed")
	}

	return apiPkg.Success(c, resp)
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return apiPkg.Unauthorized(c, "user not found in context")
	}

	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get user")
	}

	return apiPkg.Success(c, user.ToInfo())
}
