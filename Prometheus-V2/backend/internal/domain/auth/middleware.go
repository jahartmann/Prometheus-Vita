package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// claimsContextKey is the echo.Context key under which RequireAuth stores the
// verified access-token claims. It is package-private; other packages must go
// through ClaimsFromContext to read the claims rather than reaching for a
// string key directly.
const claimsContextKey = "auth.claims"

// ClaimsFromContext returns the access-token claims set by RequireAuth, or
// (nil, false) if the request was not authenticated. Handlers in other
// domains use this instead of touching c.Get directly so the storage key
// can be changed in one place.
func ClaimsFromContext(c echo.Context) (*AccessClaims, bool) {
	v := c.Get(claimsContextKey)
	if v == nil {
		return nil, false
	}
	claims, ok := v.(*AccessClaims)
	return claims, ok
}

// RequireAuth returns a middleware that rejects requests without a valid
// Authorization: Bearer <jwt> header. The decoded claims land in the Echo
// context under claimsContextKey.
func RequireAuth(signer *JWTSigner) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Request().Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(h, prefix) {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
			}
			claims, err := signer.VerifyAccessToken(strings.TrimPrefix(h, prefix))
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			c.Set(claimsContextKey, claims)
			return next(c)
		}
	}
}

// RequirePermission returns a middleware that rejects requests whose claims
// do not contain the required permission. Must be chained AFTER RequireAuth.
func RequirePermission(perm string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := ClaimsFromContext(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
			}
			if !HasPermission(claims.Permissions, perm) {
				return echo.NewHTTPError(http.StatusForbidden, "permission denied")
			}
			return next(c)
		}
	}
}
