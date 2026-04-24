package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/backup"
	"github.com/antigravity/prometheus/internal/service/loganalyzer"
)

type LogReportScheduleJob struct {
	scheduleRepo repository.LogReportScheduleRepository
	reporter     *loganalyzer.Reporter
	interval     time.Duration
}

func NewLogReportScheduleJob(scheduleRepo repository.LogReportScheduleRepository, reporter *loganalyzer.Reporter, interval time.Duration) *LogReportScheduleJob {
	return &LogReportScheduleJob{scheduleRepo: scheduleRepo, reporter: reporter, interval: interval}
}

func (j *LogReportScheduleJob) Name() string {
	return "log_report_schedule"
}

func (j *LogReportScheduleJob) Interval() time.Duration {
	return j.interval
}

func (j *LogReportScheduleJob) Run(ctx context.Context) error {
	now := time.Now()
	schedules, err := j.scheduleRepo.GetDue(ctx, now)
	if err != nil {
		return fmt.Errorf("list due log report schedules: %w", err)
	}
	for _, schedule := range schedules {
		j.processSchedule(ctx, schedule, now)
	}
	return nil
}

func (j *LogReportScheduleJob) processSchedule(ctx context.Context, schedule model.LogReportSchedule, now time.Time) {
	if len(schedule.NodeIDs) == 0 {
		slog.Warn("skipping log report schedule without nodes", slog.String("schedule_id", schedule.ID.String()))
		return
	}
	windowHours := schedule.TimeWindowHours
	if windowHours <= 0 {
		windowHours = 24
	}
	base := now
	if schedule.NextRunAt != nil {
		base = *schedule.NextRunAt
	}
	nextRun, err := backup.NextRun(schedule.CronExpression, base)
	if err != nil {
		slog.Error("failed to calculate next log report run",
			slog.String("schedule_id", schedule.ID.String()),
			slog.String("cron", schedule.CronExpression),
			slog.Any("error", err),
		)
		return
	}
	schedule.LastRunAt = &now
	schedule.NextRunAt = &nextRun
	if err := j.scheduleRepo.Update(ctx, &schedule); err != nil {
		slog.Error("failed to advance log report schedule",
			slog.String("schedule_id", schedule.ID.String()),
			slog.Any("error", err),
		)
		return
	}

	req := model.AnalyzeLogsRequest{
		NodeIDs:  schedule.NodeIDs,
		TimeFrom: now.Add(-time.Duration(windowHours) * time.Hour),
		TimeTo:   now,
		Context:  fmt.Sprintf("Geplanter Log-Report %s", schedule.ID.String()),
	}
	if _, err := j.reporter.AnalyzeScheduled(ctx, req, schedule.ID); err != nil {
		slog.Error("scheduled log report failed",
			slog.String("schedule_id", schedule.ID.String()),
			slog.Any("error", err),
		)
		return
	}
	slog.Info("scheduled log report completed", slog.String("schedule_id", schedule.ID.String()))
}
