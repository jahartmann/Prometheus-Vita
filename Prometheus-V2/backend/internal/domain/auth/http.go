package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/antigravity/prometheus-v2/internal/api"
	"github.com/labstack/echo/v4"
)

const refreshCookieName = "prometheus_v2_refresh"

// HTTPHandler wires HTTP handlers for the auth domain. Construct via
// NewHTTPHandler. The Reader behaviors are exposed by the underlying
// Service through this handler's Login/Refresh/Logout/GetMe methods.
type HTTPHandler struct {
	service      *Service
	cookieDomain string
	cookieSecure bool
}

// NewHTTPHandler constructs a handler. cookieDomain is the Domain attribute
// for the refresh cookie (empty string scopes to the request host); cookieSecure
// gates the Secure attribute (true in production, typically false for
// localhost dev over plain HTTP).
func NewHTTPHandler(s *Service, cookieDomain string, cookieSecure bool) *HTTPHandler {
	return &HTTPHandler{service: s, cookieDomain: cookieDomain, cookieSecure: cookieSecure}
}

// Login validates credentials, mints a fresh token pair, sets the refresh
// cookie, and returns the access token + user in the JSON body.
func (h *HTTPHandler) Login(c echo.Context) error {
	var req api.Credentials
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	tokens, err := h.service.Login(c.Request().Context(), LoginRequest{
		Email:     string(req.Email),
		Password:  req.Password,
		UserAgent: c.Request().UserAgent(),
		IPAddress: c.RealIP(),
	})
	if err != nil {
		return mapAuthError(err)
	}
	h.setRefreshCookie(c, tokens.RefreshToken, tokens.RefreshExpiry)
	return c.JSON(http.StatusOK, tokensToResponse(tokens))
}

// Refresh rotates the refresh token bound to the cookie. On any failure the
// cookie is cleared so a stale value cannot keep producing 401s.
func (h *HTTPHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "no refresh cookie")
	}
	tokens, err := h.service.Refresh(c.Request().Context(), cookie.Value, c.Request().UserAgent(), c.RealIP())
	if err != nil {
		h.clearRefreshCookie(c)
		return mapAuthError(err)
	}
	h.setRefreshCookie(c, tokens.RefreshToken, tokens.RefreshExpiry)
	return c.JSON(http.StatusOK, tokensToResponse(tokens))
}

// Logout revokes the session bound to the cookie if present, then clears
// the cookie. Always returns 204: idempotent and never leaks whether a
// session existed.
func (h *HTTPHandler) Logout(c echo.Context) error {
	cookie, err := c.Cookie(refreshCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.service.Logout(c.Request().Context(), cookie.Value)
	}
	h.clearRefreshCookie(c)
	return c.NoContent(http.StatusNoContent)
}

// GetMe returns the currently authenticated user. RequireAuth must have run
// upstream; otherwise we fail closed with 401.
func (h *HTTPHandler) GetMe(c echo.Context) error {
	claims, ok := ClaimsFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "no claims")
	}
	u, err := h.service.GetUser(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}
	return c.JSON(http.StatusOK, userToAPI(*u))
}

func (h *HTTPHandler) setRefreshCookie(c echo.Context, token string, exp time.Time) {
	c.SetCookie(&http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		Expires:  exp,
		Domain:   h.cookieDomain,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *HTTPHandler) clearRefreshCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		Domain:   h.cookieDomain,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

// mapAuthError translates service-level sentinel errors into HTTP responses.
// All credential/session failures collapse to a single 401 message to avoid
// leaking which step failed (no user enumeration, no session probing).
func mapAuthError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrSessionNotFound),
		errors.Is(err, ErrSessionExpired),
		errors.Is(err, ErrUserNotFound):
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, ErrUserDisabled):
		return echo.NewHTTPError(http.StatusForbidden, "user disabled")
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "auth failure")
	}
}

func tokensToResponse(t *AuthTokens) api.AuthResponse {
	return api.AuthResponse{
		AccessToken:     t.AccessToken,
		AccessExpiresAt: t.AccessExpiry,
		User:            userToAPI(t.User),
	}
}

func userToAPI(u User) api.User {
	return api.User{
		Id:      u.ID,
		Email:   u.Email,
		Name:    u.Name,
		Role:    api.UserRole(u.Role),
		Enabled: u.Enabled,
	}
}
