package handler

import (
	"context"
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
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

func (h *HealthHandler) Check(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	status := HealthStatus{
		Status:   "ok",
		Services: make(map[string]string),
	}

	httpStatus := http.StatusOK

	if err := h.db.Ping(ctx); err != nil {
		status.Services["postgres"] = "unhealthy: " + err.Error()
		status.Status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		status.Services["postgres"] = "healthy"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		status.Services["redis"] = "unhealthy: " + err.Error()
		status.Status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		status.Services["redis"] = "healthy"
	}

	return c.JSON(httpStatus, status)
}
