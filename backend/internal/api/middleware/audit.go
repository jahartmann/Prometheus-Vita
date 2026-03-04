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

			// Record audit log asynchronously
			go func() {
				entry := &model.AuditLogEntry{
					Method:     c.Request().Method,
					Path:       c.Request().URL.Path,
					StatusCode: c.Response().Status,
					IPAddress:  c.RealIP(),
					UserAgent:  c.Request().UserAgent(),
					DurationMS: int(time.Since(start).Milliseconds()),
				}

				if userID, ok := c.Get(ContextKeyUserID).(uuid.UUID); ok {
					entry.UserID = &userID
				}

				if tokenID, ok := c.Get(ContextKeyAPITokenID).(uuid.UUID); ok {
					entry.APITokenID = &tokenID
				}

				_ = auditRepo.Create(context.Background(), entry)
			}()

			return err
		}
	}
}
