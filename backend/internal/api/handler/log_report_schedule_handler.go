package handler

import (
	"errors"
	"time"

	"github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/backup"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type LogReportScheduleHandler struct {
	repo repository.LogReportScheduleRepository
}

func NewLogReportScheduleHandler(repo repository.LogReportScheduleRepository) *LogReportScheduleHandler {
	return &LogReportScheduleHandler{repo: repo}
}

func (h *LogReportScheduleHandler) Create(c echo.Context) error {
	var req model.CreateLogReportScheduleRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if req.TimeWindowHours <= 0 {
		req.TimeWindowHours = 24
	}
	schedule := &model.LogReportSchedule{
		CronExpression:     req.CronExpression,
		NodeIDs:            req.NodeIDs,
		TimeWindowHours:    req.TimeWindowHours,
		DeliveryChannelIDs: req.DeliveryChannelIDs,
		IsActive:           true,
	}
	nextRun, err := backup.NextRun(schedule.CronExpression, time.Now())
	if err != nil {
		return response.BadRequest(c, "invalid cron expression")
	}
	schedule.NextRunAt = &nextRun
	if err := h.repo.Create(c.Request().Context(), schedule); err != nil {
		return response.InternalError(c, "failed to create log report schedule")
	}
	return response.Created(c, schedule)
}

func (h *LogReportScheduleHandler) List(c echo.Context) error {
	schedules, err := h.repo.List(c.Request().Context())
	if err != nil {
		return response.InternalError(c, "failed to list log report schedules")
	}
	if schedules == nil {
		schedules = []model.LogReportSchedule{}
	}
	return response.Success(c, schedules)
}

func (h *LogReportScheduleHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}

	var req model.UpdateLogReportScheduleRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	schedule, err := h.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.NotFound(c, "schedule not found")
		}
		return response.InternalError(c, "failed to get schedule")
	}

	if req.CronExpression != nil {
		schedule.CronExpression = *req.CronExpression
		nextRun, err := backup.NextRun(schedule.CronExpression, time.Now())
		if err != nil {
			return response.BadRequest(c, "invalid cron expression")
		}
		schedule.NextRunAt = &nextRun
	}
	if req.NodeIDs != nil {
		schedule.NodeIDs = *req.NodeIDs
	}
	if req.TimeWindowHours != nil {
		schedule.TimeWindowHours = *req.TimeWindowHours
	}
	if req.DeliveryChannelIDs != nil {
		schedule.DeliveryChannelIDs = *req.DeliveryChannelIDs
	}
	if req.IsActive != nil {
		schedule.IsActive = *req.IsActive
	}

	if err := h.repo.Update(c.Request().Context(), schedule); err != nil {
		return response.InternalError(c, "failed to update schedule")
	}
	return response.Success(c, schedule)
}

func (h *LogReportScheduleHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}
	if err := h.repo.Delete(c.Request().Context(), id); err != nil {
		return response.InternalError(c, "failed to delete schedule")
	}
	return response.NoContent(c)
}
