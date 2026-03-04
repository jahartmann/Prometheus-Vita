package handler

import (
	"errors"
	"fmt"
	"io"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/backup"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// BackupHandler exposes HTTP endpoints for creating, listing, downloading,
// restoring, and diffing configuration backups.
type BackupHandler struct {
	service        *backup.Service
	restoreService *backup.RestoreService
}

// NewBackupHandler creates a new BackupHandler with references to both the
// main backup service and the restore service.
func NewBackupHandler(service *backup.Service, restoreService *backup.RestoreService) *BackupHandler {
	return &BackupHandler{
		service:        service,
		restoreService: restoreService,
	}
}

// CreateBackup handles POST /nodes/:id/backup.
// It triggers a new configuration backup for the specified node.
func (h *BackupHandler) CreateBackup(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	var req model.CreateBackupRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	result, err := h.service.CreateBackup(c.Request().Context(), nodeID, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.InternalError(c, "failed to create backup")
	}

	return apiPkg.Created(c, result)
}

// ListBackups handles GET /nodes/:id/backups.
// It returns all backups for the specified node.
func (h *BackupHandler) ListBackups(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	backups, err := h.service.ListBackups(c.Request().Context(), nodeID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list backups")
	}

	return apiPkg.Success(c, backups)
}

// GetBackup handles GET /backups/:id.
// It returns a single backup by its ID.
func (h *BackupHandler) GetBackup(c echo.Context) error {
	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid backup id")
	}

	result, err := h.service.GetBackup(c.Request().Context(), backupID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "backup not found")
		}
		return apiPkg.InternalError(c, "failed to get backup")
	}

	return apiPkg.Success(c, result)
}

// GetBackupFiles handles GET /backups/:id/files.
// It returns all file metadata for a backup (without file content).
func (h *BackupHandler) GetBackupFiles(c echo.Context) error {
	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid backup id")
	}

	files, err := h.service.GetBackupFiles(c.Request().Context(), backupID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get backup files")
	}

	return apiPkg.Success(c, files)
}

// GetBackupFile handles GET /backups/:id/files/*.
// It returns a single file (with content) from a backup, identified by the
// file path in the wildcard portion of the URL.
func (h *BackupHandler) GetBackupFile(c echo.Context) error {
	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid backup id")
	}

	filePath := c.Param("*")
	if filePath == "" {
		return apiPkg.BadRequest(c, "file path is required")
	}
	// Ensure the path starts with /
	if filePath[0] != '/' {
		filePath = "/" + filePath
	}

	file, err := h.service.GetBackupFile(c.Request().Context(), backupID, filePath)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "backup file not found")
		}
		return apiPkg.InternalError(c, "failed to get backup file")
	}

	return apiPkg.Success(c, file)
}

// DiffBackup handles GET /backups/:id/diff.
// It computes and returns the file-level diff between this backup and its
// predecessor.
func (h *BackupHandler) DiffBackup(c echo.Context) error {
	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid backup id")
	}

	diffs, err := h.service.DiffBackup(c.Request().Context(), backupID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "backup not found")
		}
		return apiPkg.InternalError(c, "failed to compute backup diff")
	}

	return apiPkg.Success(c, diffs)
}

// DeleteBackup handles DELETE /backups/:id.
// It removes a backup and all its associated files.
func (h *BackupHandler) DeleteBackup(c echo.Context) error {
	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid backup id")
	}

	if err := h.service.DeleteBackup(c.Request().Context(), backupID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "backup not found")
		}
		return apiPkg.InternalError(c, "failed to delete backup")
	}

	return apiPkg.NoContent(c)
}

// RestoreBackup handles POST /backups/:id/restore.
// It restores the specified files from a backup to the originating node.
// Supports dry-run mode for previewing changes without applying them.
func (h *BackupHandler) RestoreBackup(c echo.Context) error {
	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid backup id")
	}

	var req model.RestoreRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if len(req.FilePaths) == 0 {
		return apiPkg.BadRequest(c, "file_paths is required")
	}

	result, err := h.restoreService.RestoreFiles(c.Request().Context(), backupID, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "backup not found")
		}
		return apiPkg.InternalError(c, "failed to restore backup")
	}

	return apiPkg.Success(c, result)
}

// DownloadBackup handles GET /backups/:id/download.
// It generates and streams a tar.gz archive of all files in the backup.
func (h *BackupHandler) DownloadBackup(c echo.Context) error {
	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid backup id")
	}

	reader, err := h.restoreService.GenerateArchive(c.Request().Context(), backupID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "backup not found")
		}
		return apiPkg.InternalError(c, "failed to generate backup archive")
	}

	c.Response().Header().Set("Content-Type", "application/gzip")
	c.Response().Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"backup-%s.tar.gz\"", backupID.String()))

	_, err = io.Copy(c.Response().Writer, reader)
	return err
}
