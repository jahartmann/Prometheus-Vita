package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type RateLimitConfig struct {
	RequestsPerMinute int
	Enabled           bool
}

func RateLimit(redisClient *redis.Client, cfg RateLimitConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !cfg.Enabled || redisClient == nil {
				return next(c)
			}

			// Use IP + user for rate limit key
			ip := c.RealIP()
			key := fmt.Sprintf("ratelimit:%s", ip)

			// If user is authenticated, use user-specific key
			if userID, ok := c.Get(ContextKeyUserID).(fmt.Stringer); ok {
				key = fmt.Sprintf("ratelimit:user:%s", userID.String())
			}

			ctx := context.Background()
			now := time.Now()
			windowStart := now.Add(-1 * time.Minute)

			pipe := redisClient.Pipeline()
			// Remove old entries
			pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
			// Add current request
			pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})
			// Count requests in window
			countCmd := pipe.ZCard(ctx, key)
			// Set expiry
			pipe.Expire(ctx, key, 2*time.Minute)

			if _, err := pipe.Exec(ctx); err != nil {
				// On Redis error, allow request through
				return next(c)
			}

			count := countCmd.Val()
			if count > int64(cfg.RequestsPerMinute) {
				c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RequestsPerMinute))
				c.Response().Header().Set("X-RateLimit-Remaining", "0")
				c.Response().Header().Set("Retry-After", "60")
				return response.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded")
			}

			remaining := int64(cfg.RequestsPerMinute) - count
			c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RequestsPerMinute))
			c.Response().Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			return next(c)
		}
	}
}
