package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
)

func Recovery() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					stackTrace := string(buf[:n])

					slog.Error("panic recovered",
						slog.Any("panic", r),
						slog.String("stack", stackTrace),
						slog.String("method", c.Request().Method),
						slog.String("uri", c.Request().RequestURI),
					)

					err = c.JSON(http.StatusInternalServerError, map[string]any{
						"success": false,
						"error":   fmt.Sprintf("internal server error: %v", r),
					})
				}
			}()
			return next(c)
		}
	}
}
