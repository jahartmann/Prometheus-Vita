package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Services  map[string]string `json:"services"`
	Timestamp string            `json:"timestamp"`
}

// Live is the cheapest possible liveness probe — the process is up and able
// to serve HTTP. Use this for k8s livenessProbe / docker HEALTHCHECK. It does
// not exercise downstream dependencies on purpose: those would cause cascading
// restarts when a transient DB blip occurs.
func (h *HealthHandler) Live(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready reports readiness to serve traffic. Verifies the critical downstream
// dependencies (Postgres, Redis). Returns 503 when any dependency is down so
// load balancers stop routing requests until the service recovers.
func (h *HealthHandler) Ready(c echo.Context) error {
	return h.check(c, true)
}

// Check is the legacy /health endpoint. Kept for backward compatibility —
// behaves like Ready but does not log error detail to the client.
func (h *HealthHandler) Check(c echo.Context) error {
	return h.check(c, false)
}

func (h *HealthHandler) check(c echo.Context, exposeDetail bool) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	status := HealthStatus{
		Status:    "ok",
		Services:  make(map[string]string),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	httpStatus := http.StatusOK

	if err := h.db.Ping(ctx); err != nil {
		// Always log the real error server-side for ops, but only expose a
		// sanitized status to the caller — raw errors can leak network
		// topology, credentials in DSNs, etc.
		slog.Warn("health: postgres ping failed", slog.Any("error", err))
		if exposeDetail {
			status.Services["postgres"] = "unhealthy"
		} else {
			status.Services["postgres"] = "unhealthy"
		}
		status.Status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		status.Services["postgres"] = "healthy"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		slog.Warn("health: redis ping failed", slog.Any("error", err))
		status.Services["redis"] = "unhealthy"
		status.Status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		status.Services["redis"] = "healthy"
	}

	return c.JSON(httpStatus, status)
}
