package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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
			if statusCode == 0 {
				if err != nil {
					statusCode = http.StatusInternalServerError
				} else {
					statusCode = http.StatusOK
				}
			}
			ip := c.RealIP()
			userAgent := c.Request().UserAgent()
			duration := int(time.Since(start).Milliseconds())
			userID, hasUser := c.Get(ContextKeyUserID).(uuid.UUID)
			tokenID, hasToken := c.Get(ContextKeyAPITokenID).(uuid.UUID)
			auditMeta := classifyAuditEvent(method, path, statusCode)

			// Record audit log asynchronously
			go func(uid uuid.UUID, hasU bool, tid uuid.UUID, hasT bool) {
				entry := &model.AuditLogEntry{
					Method:     method,
					Path:       path,
					StatusCode: statusCode,
					IPAddress:  ip,
					UserAgent:  userAgent,
					DurationMS: duration,
				}
				if auditMeta != nil {
					if raw, marshalErr := json.Marshal(auditMeta); marshalErr == nil {
						entry.RequestBody = raw
					}
				}

				if hasU {
					entry.UserID = &uid
				}

				if hasT {
					entry.APITokenID = &tid
				}

				_ = auditRepo.Create(context.Background(), entry)
			}(userID, hasUser, tokenID, hasToken)

			return err
		}
	}
}

type auditEventMetadata struct {
	Critical bool   `json:"critical"`
	Category string `json:"category"`
	Action   string `json:"action"`
	Risk     string `json:"risk"`
	Success  bool   `json:"success"`
}

type auditRouteClass struct {
	Prefix   string
	Category string
	Risk     string
}

var criticalAuditRoutes = []auditRouteClass{
	{Prefix: "/api/v1/auth/login", Category: "auth", Risk: "medium"},
	{Prefix: "/api/v1/auth/logout", Category: "auth", Risk: "low"},
	{Prefix: "/api/v1/users", Category: "users", Risk: "high"},
	{Prefix: "/api/v1/password-policy", Category: "security", Risk: "high"},
	{Prefix: "/api/v1/permissions", Category: "permissions", Risk: "high"},
	{Prefix: "/api/v1/vm-permissions", Category: "permissions", Risk: "high"},
	{Prefix: "/api/v1/vm-groups", Category: "permissions", Risk: "medium"},
	{Prefix: "/api/v1/gateway/tokens", Category: "api_tokens", Risk: "high"},
	{Prefix: "/api/v1/nodes", Category: "infrastructure", Risk: "high"},
	{Prefix: "/api/v1/backups", Category: "backup", Risk: "high"},
	{Prefix: "/api/v1/backup-schedules", Category: "backup", Risk: "medium"},
	{Prefix: "/api/v1/dr", Category: "disaster_recovery", Risk: "high"},
	{Prefix: "/api/v1/migrations", Category: "migration", Risk: "high"},
	{Prefix: "/api/v1/approvals", Category: "agent_approval", Risk: "high"},
	{Prefix: "/api/v1/agent/config", Category: "agent_config", Risk: "high"},
	{Prefix: "/api/v1/reflexes", Category: "automation", Risk: "high"},
	{Prefix: "/api/v1/ssh-keys", Category: "ssh_keys", Risk: "high"},
	{Prefix: "/api/v1/logs", Category: "logs", Risk: "medium"},
	{Prefix: "/api/v1/log-bookmarks", Category: "logs", Risk: "low"},
	{Prefix: "/api/v1/log-anomalies", Category: "logs", Risk: "medium"},
	{Prefix: "/api/v1/network-scans", Category: "security", Risk: "medium"},
	{Prefix: "/api/v1/network-devices", Category: "security", Risk: "medium"},
	{Prefix: "/api/v1/network-anomalies", Category: "security", Risk: "medium"},
	{Prefix: "/api/v1/scan-baselines", Category: "security", Risk: "medium"},
	{Prefix: "/api/v1/security", Category: "security", Risk: "high"},
	{Prefix: "/api/v1/environments", Category: "settings", Risk: "medium"},
	{Prefix: "/api/v1/notifications", Category: "settings", Risk: "medium"},
	{Prefix: "/api/v1/alerts", Category: "security", Risk: "medium"},
	{Prefix: "/api/v1/escalation", Category: "security", Risk: "medium"},
	{Prefix: "/api/v1/tags", Category: "metadata", Risk: "low"},
	{Prefix: "/api/v1/chat", Category: "agent", Risk: "medium"},
	{Prefix: "/api/v1/brain", Category: "agent_knowledge", Risk: "medium"},
	{Prefix: "/api/v1/drift", Category: "configuration", Risk: "medium"},
}

func classifyAuditEvent(method, path string, statusCode int) *auditEventMetadata {
	action := auditAction(method)
	if action == "read" {
		return nil
	}

	normalizedPath := strings.TrimRight(path, "/")
	if strings.Contains(normalizedPath, "/vms/") && action != "read" {
		return newAuditMetadata("vm", action, "high", statusCode)
	}

	for _, route := range criticalAuditRoutes {
		if normalizedPath == route.Prefix || strings.HasPrefix(normalizedPath, route.Prefix+"/") {
			return newAuditMetadata(route.Category, action, route.Risk, statusCode)
		}
	}

	return newAuditMetadata("api", action, "low", statusCode)
}

func newAuditMetadata(category, action, risk string, statusCode int) *auditEventMetadata {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return &auditEventMetadata{
		Critical: risk == "high",
		Category: category,
		Action:   action,
		Risk:     risk,
		Success:  statusCode >= 200 && statusCode < 400,
	}
}

func auditAction(method string) string {
	switch strings.ToUpper(method) {
	case http.MethodPost:
		return "create_or_execute"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "read"
	}
}
