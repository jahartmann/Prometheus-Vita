package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			req := c.Request()
			res := c.Response()

			attrs := []slog.Attr{
				slog.String("method", req.Method),
				slog.String("uri", req.RequestURI),
				slog.Int("status", res.Status),
				slog.Duration("latency", time.Since(start)),
				slog.String("remote_ip", c.RealIP()),
				slog.String("request_id", res.Header().Get(echo.HeaderXRequestID)),
			}

			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
			}

			level := slog.LevelInfo
			if res.Status >= 500 {
				level = slog.LevelError
			} else if res.Status >= 400 {
				level = slog.LevelWarn
			}

			slog.LogAttrs(req.Context(), level, "request",
				attrs...,
			)

			return err
		}
	}
}
