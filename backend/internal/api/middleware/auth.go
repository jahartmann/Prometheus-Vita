package middleware

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/auth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	ContextKeyUserID   = "user_id"
	ContextKeyUsername = "username"
	ContextKeyRole     = "role"
)

// activeUserCacheTTL bounds how long a user.IsActive lookup is cached in
// memory. Short enough that a deactivated user loses access within the TTL
// even without a server restart; long enough that we don't add a DB roundtrip
// to every single authenticated request.
const activeUserCacheTTL = 30 * time.Second

type activeUserCacheEntry struct {
	isActive bool
	expires  time.Time
}

var (
	activeUserCacheMu sync.RWMutex
	activeUserCache   = map[uuid.UUID]activeUserCacheEntry{}
)

// JWTAuth validates the bearer token and rejects requests from users who have
// been deactivated since their token was issued. The IsActive check is cached
// for activeUserCacheTTL to keep the per-request cost low; deactivation
// becomes effective within that window without requiring a server restart or
// JWT secret rotation. userRepo may be nil for legacy callers — in that
// case the IsActive check is skipped.
func JWTAuth(jwtSvc *auth.JWTService, userRepo repository.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip if already authenticated via API key.
			if c.Get(ContextKeyUserID) != nil {
				return next(c)
			}

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Unauthorized(c, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return response.Unauthorized(c, "invalid authorization header format")
			}

			claims, err := jwtSvc.ValidateAccessToken(parts[1])
			if err != nil {
				return response.Unauthorized(c, "invalid or expired token")
			}

			if userRepo != nil {
				if !checkUserActiveCached(c.Request().Context(), userRepo, claims.UserID) {
					return response.Unauthorized(c, "account is inactive")
				}
			}

			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyUsername, claims.Username)
			c.Set(ContextKeyRole, claims.Role)

			return next(c)
		}
	}
}

func checkUserActiveCached(ctx context.Context, userRepo repository.UserRepository, id uuid.UUID) bool {
	activeUserCacheMu.RLock()
	entry, ok := activeUserCache[id]
	activeUserCacheMu.RUnlock()
	if ok && time.Now().Before(entry.expires) {
		return entry.isActive
	}

	user, err := userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		// Fail closed: if we can't determine status, treat as inactive.
		setActiveCache(id, false)
		return false
	}
	setActiveCache(id, user.IsActive)
	return user.IsActive
}

func setActiveCache(id uuid.UUID, active bool) {
	activeUserCacheMu.Lock()
	activeUserCache[id] = activeUserCacheEntry{
		isActive: active,
		expires:  time.Now().Add(activeUserCacheTTL),
	}
	activeUserCacheMu.Unlock()
}

// InvalidateActiveUserCache lets the user service punch a hole in the cache
// when an admin deactivates a user, so the change takes effect immediately
// for in-flight tokens (rather than waiting up to activeUserCacheTTL).
func InvalidateActiveUserCache(id uuid.UUID) {
	activeUserCacheMu.Lock()
	delete(activeUserCache, id)
	activeUserCacheMu.Unlock()
}
