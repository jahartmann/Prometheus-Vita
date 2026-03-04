package handler

import (
	"errors"
	"time"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/backup"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ScheduleHandler exposes HTTP endpoints for managing backup schedules
// (CRUD operations with cron validation).
type ScheduleHandler struct {
	scheduleRepo repository.ScheduleRepository
	cronParser   func(string) error
}

// NewScheduleHandler creates a new ScheduleHandler. It uses backup.ParseCron
// as the default cron expression validator.
func NewScheduleHandler(scheduleRepo repository.ScheduleRepository) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleRepo: scheduleRepo,
		cronParser:   backup.ParseCron,
	}
}

// CreateSchedule handles POST /nodes/:id/backup-schedules.
// It validates the cron expression, calculates the initial next run time,
// and persists the new schedule.
func (h *ScheduleHandler) CreateSchedule(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	var req model.CreateScheduleRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if req.CronExpression == "" {
		return apiPkg.BadRequest(c, "cron_expression is required")
	}

	if err := h.cronParser(req.CronExpression); err != nil {
		return apiPkg.BadRequest(c, "invalid cron expression: "+err.Error())
	}

	retentionCount := req.RetentionCount
	if retentionCount <= 0 {
		retentionCount = 10 // default retention
	}

	// Calculate next run time
	nextRun, err := backup.NextRun(req.CronExpression, time.Now())
	if err != nil {
		return apiPkg.BadRequest(c, "failed to calculate next run time: "+err.Error())
	}

	schedule := &model.BackupSchedule{
		NodeID:         nodeID,
		CronExpression: req.CronExpression,
		IsActive:       req.IsActive,
		RetentionCount: retentionCount,
		NextRunAt:      &nextRun,
	}

	if err := h.scheduleRepo.Create(c.Request().Context(), schedule); err != nil {
		return apiPkg.InternalError(c, "failed to create schedule")
	}

	return apiPkg.Created(c, schedule)
}

// ListSchedules handles GET /nodes/:id/backup-schedules.
// It returns all backup schedules for the specified node.
func (h *ScheduleHandler) ListSchedules(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid node id")
	}

	schedules, err := h.scheduleRepo.ListByNode(c.Request().Context(), nodeID)
	if err != nil {
		return apiPkg.InternalError(c, "failed to list schedules")
	}

	return apiPkg.Success(c, schedules)
}

// UpdateSchedule handles PUT /backup-schedules/:id.
// It allows partial updates to the cron expression, active state, and
// retention count. If the cron expression changes, the next run time is
// recalculated.
func (h *ScheduleHandler) UpdateSchedule(c echo.Context) error {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid schedule id")
	}

	var req model.UpdateScheduleRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	schedule, err := h.scheduleRepo.GetByID(c.Request().Context(), scheduleID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "schedule not found")
		}
		return apiPkg.InternalError(c, "failed to get schedule")
	}

	if req.CronExpression != nil {
		if err := h.cronParser(*req.CronExpression); err != nil {
			return apiPkg.BadRequest(c, "invalid cron expression: "+err.Error())
		}
		schedule.CronExpression = *req.CronExpression

		// Recalculate next run
		nextRun, err := backup.NextRun(*req.CronExpression, time.Now())
		if err != nil {
			return apiPkg.BadRequest(c, "failed to calculate next run time: "+err.Error())
		}
		schedule.NextRunAt = &nextRun
	}

	if req.IsActive != nil {
		schedule.IsActive = *req.IsActive
	}

	if req.RetentionCount != nil {
		schedule.RetentionCount = *req.RetentionCount
	}

	if err := h.scheduleRepo.Update(c.Request().Context(), schedule); err != nil {
		return apiPkg.InternalError(c, "failed to update schedule")
	}

	return apiPkg.Success(c, schedule)
}

// DeleteSchedule handles DELETE /backup-schedules/:id.
// It removes a backup schedule by its ID.
func (h *ScheduleHandler) DeleteSchedule(c echo.Context) error {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid schedule id")
	}

	if err := h.scheduleRepo.Delete(c.Request().Context(), scheduleID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "schedule not found")
		}
		return apiPkg.InternalError(c, "failed to delete schedule")
	}

	return apiPkg.NoContent(c)
}
