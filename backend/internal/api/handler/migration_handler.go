package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/migration"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type MigrationHandler struct {
	service *migration.Service
}

func NewMigrationHandler(service *migration.Service) *MigrationHandler {
	return &MigrationHandler{service: service}
}

// Start handles POST /api/v1/migrations
func (h *MigrationHandler) Start(c echo.Context) error {
	var req model.StartMigrationRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if req.SourceNodeID == uuid.Nil || req.TargetNodeID == uuid.Nil {
		return apiPkg.BadRequest(c, "source_node_id and target_node_id are required")
	}
	if req.VMID <= 0 {
		return apiPkg.BadRequest(c, "vmid is required")
	}
	if req.TargetStorage == "" {
		return apiPkg.BadRequest(c, "target_storage is required")
	}

	// Get user ID from context
	var userID *uuid.UUID
	if uid, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID); ok {
		userID = &uid
	}

	result, err := h.service.StartMigration(c.Request().Context(), req, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "node not found")
		}
		return apiPkg.BadRequest(c, err.Error())
	}

	return apiPkg.Created(c, result.ToResponse())
}

// List handles GET /api/v1/migrations
func (h *MigrationHandler) List(c echo.Context) error {
	migrations, err := h.service.ListFilteredMigrations(c.Request().Context(), ParseQueryFilter(c))
	if err != nil {
		return apiPkg.InternalError(c, "failed to list migrations")
	}

	responses := make([]model.MigrationResponse, len(migrations))
	for i, m := range migrations {
		responses[i] = m.ToResponse()
	}
	return apiPkg.Success(c, responses)
}

// Get handles GET /api/v1/migrations/:id
func (h *MigrationHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid migration id")
	}

	m, err := h.service.GetMigration(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "migration not found")
		}
		return apiPkg.InternalError(c, "failed to get migration")
	}

	return apiPkg.Success(c, m.ToResponse())
}

// Cancel handles POST /api/v1/migrations/:id/cancel
func (h *MigrationHandler) Cancel(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid migration id")
	}

	// Extract caller identity for ownership check
	var userID *uuid.UUID
	if uid, ok := c.Get(middleware.ContextKeyUserID).(uuid.UUID); ok {
		userID = &uid
	}
	role, _ := c.Get(middleware.ContextKeyRole).(model.UserRole)
	isAdmin := role == model.RoleAdmin

	if err := h.service.CancelMigration(c.Request().Context(), id, userID, isAdmin); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "migration not found")
		}
		return apiPkg.BadRequest(c, err.Error())
	}

	return apiPkg.Success(c, map[string]string{"status": "cancelled"})
}

// Delete handles DELETE /api/v1/migrations/:id
func (h *MigrationHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid migration id")
	}

	if err := h.service.DeleteMigration(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "migration not found")
		}
		return apiPkg.BadRequest(c, err.Error())
	}

	return apiPkg.NoContent(c)
}

// ListByNode handles GET /api/v1/nodes/:id/migrations
func (h *MigrationHandler) ListByNode(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	filter := ParseQueryFilter(c)
	filter.NodeID = &nodeID
	migrations, err := h.service.ListFilteredMigrations(c.Request().Context(), filter)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list migrations")
	}

	responses := make([]model.MigrationResponse, len(migrations))
	for i, m := range migrations {
		responses[i] = m.ToResponse()
	}
	return apiPkg.Success(c, responses)
}

// GetLogs handles GET /api/v1/migrations/:id/logs
func (h *MigrationHandler) GetLogs(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid migration id")
	}

	logs, err := h.service.GetLogs(c.Request().Context(), id)
	if err != nil {
		return apiPkg.InternalError(c, "failed to get migration logs")
	}
	if logs == nil {
		logs = []repository.MigrationLog{}
	}

	return apiPkg.Success(c, logs)
}
