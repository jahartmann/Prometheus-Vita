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
		slog.Warn("CORS_ALLOWED_ORIGINS not set, defaulting to localhost only")
		origins = []string{"http://localhost:3000", "http://localhost:8080"}
	}
	return echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     origins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		AllowCredentials: len(origins) > 0 && origins[0] != "*",
		MaxAge:           86400,
	})
}
