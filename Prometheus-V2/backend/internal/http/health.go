package http

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type RedisPinger interface {
	Ping(ctx context.Context) error
}

func RegisterHealth(e *echo.Echo, db DBPinger, redis RedisPinger) {
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.GET("/readyz", func(c echo.Context) error {
		ctx := c.Request().Context()
		if db == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db not initialized"})
		}
		if err := db.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db unhealthy", "error": err.Error()})
		}
		if redis != nil {
			if err := redis.Ping(ctx); err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "redis unhealthy", "error": err.Error()})
			}
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
}
