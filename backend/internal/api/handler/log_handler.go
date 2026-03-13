package handler

import (
	"fmt"
	"log/slog"

	"github.com/antigravity/prometheus/internal/apierror"
	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	nodeService "github.com/antigravity/prometheus/internal/service/node"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type LogHandler struct {
	nodeSvc *nodeService.Service
}

func NewLogHandler(nodeSvc *nodeService.Service) *LogHandler {
	return &LogHandler{nodeSvc: nodeSvc}
}

// GetLogs returns the last N lines of a system log file.
// Tries Proxmox API first (syslog endpoint), falls back to SSH.
func (h *LogHandler) GetLogs(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	logFile := c.QueryParam("file")
	if logFile == "" {
		logFile = "syslog"
	}

	// Whitelist allowed log files for security
	allowedLogs := map[string]string{
		"syslog":       "/var/log/syslog",
		"auth":         "/var/log/auth.log",
		"pveproxy":     "/var/log/pveproxy/access.log",
		"pvedaemon":    "/var/log/pvedaemon.log",
		"pve-firewall": "/var/log/pve-firewall.log",
		"corosync":     "/var/log/corosync/corosync.log",
		"tasks":        "/var/log/pve/tasks/active",
	}

	logPath, ok := allowedLogs[logFile]
	if !ok {
		return apiPkg.BadRequest(c, "unsupported log file")
	}

	lines := 100
	if l := c.QueryParam("lines"); l != "" {
		fmt.Sscanf(l, "%d", &lines)
		if lines < 1 {
			lines = 1
		}
		if lines > 1000 {
			lines = 1000
		}
	}

	// Try Proxmox API first for syslog
	if logFile == "syslog" {
		logLines, apiErr := h.nodeSvc.GetNodeSyslog(c.Request().Context(), nodeID, lines)
		if apiErr == nil {
			return apiPkg.Success(c, map[string]interface{}{
				"file":  logFile,
				"path":  logPath,
				"lines": logLines,
			})
		}
		slog.Warn("proxmox api syslog failed, trying ssh",
			slog.String("node_id", nodeID.String()), slog.Any("error", apiErr))
	}

	// Fallback: SSH
	cmd := fmt.Sprintf("tail -n %d %s 2>/dev/null || echo 'Log file not available'", lines, logPath)
	result, err := h.nodeSvc.RunSSHCommand(c.Request().Context(), nodeID, cmd)
	if err != nil {
		slog.Error("failed to read logs via ssh",
			slog.String("node_id", nodeID.String()),
			slog.String("log_file", logFile),
			slog.Any("error", err))
		return apiPkg.FromAPIError(c, apierror.NodeSSHFailed(err))
	}

	return apiPkg.Success(c, map[string]interface{}{
		"file":  logFile,
		"path":  logPath,
		"lines": result.Stdout,
	})
}
