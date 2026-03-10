package middleware

import (
	"github.com/labstack/echo/v4"
)

func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			res := c.Response().Header()

			res.Set("X-Content-Type-Options", "nosniff")
			res.Set("X-Frame-Options", "DENY")
			res.Set("X-XSS-Protection", "1; mode=block")
			res.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			res.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			res.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' ws: wss:; font-src 'self'")

			if c.Request().TLS != nil {
				res.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			return next(c)
		}
	}
}
