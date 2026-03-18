package middleware

import (
	"context"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func AuditLog(auditRepo repository.AuditRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Process request
			err := next(c)

			// Capture values before goroutine to avoid data race on echo.Context
			method := c.Request().Method
			path := c.Request().URL.Path
			statusCode := c.Response().Status
			ip := c.RealIP()
			userAgent := c.Request().UserAgent()
			duration := int(time.Since(start).Milliseconds())
			userID, hasUser := c.Get(ContextKeyUserID).(uuid.UUID)
			tokenID, hasToken := c.Get(ContextKeyAPITokenID).(uuid.UUID)

			// Record audit log asynchronously
			go func() {
				entry := &model.AuditLogEntry{
					Method:     method,
					Path:       path,
					StatusCode: statusCode,
					IPAddress:  ip,
					UserAgent:  userAgent,
					DurationMS: duration,
				}

				if hasUser {
					entry.UserID = &userID
				}

				if hasToken {
					entry.APITokenID = &tokenID
				}

				_ = auditRepo.Create(context.Background(), entry)
			}()

			return err
		}
	}
}
