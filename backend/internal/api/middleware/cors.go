package middleware

import (
	"log/slog"

	"github.com/antigravity/prometheus/internal/config"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

func CORS(cfg config.CORSConfig) echo.MiddlewareFunc {
	origins := cfg.AllowOrigins
	if len(origins) == 0 {
		// When not configured, allow all origins but log a warning.
		// In production, CORS_ALLOWED_ORIGINS should be set explicitly.
		slog.Warn("CORS_ALLOWED_ORIGINS not set — allowing all origins. Set CORS_ALLOWED_ORIGINS for production.")
		origins = []string{"*"}
	}
	return echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     origins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		AllowCredentials: len(origins) > 0 && origins[0] != "*",
		MaxAge:           86400,
	})
}
