package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/backup"
	"github.com/antigravity/prometheus/internal/service/loganalyzer"
	"github.com/antigravity/prometheus/internal/service/notification"
)

type LogReportScheduleJob struct {
	scheduleRepo repository.LogReportScheduleRepository
	reporter     *loganalyzer.Reporter
	notifSvc     *notification.Service
	interval     time.Duration
}

func NewLogReportScheduleJob(scheduleRepo repository.LogReportScheduleRepository, reporter *loganalyzer.Reporter, notifSvc *notification.Service, interval time.Duration) *LogReportScheduleJob {
	return &LogReportScheduleJob{scheduleRepo: scheduleRepo, reporter: reporter, notifSvc: notifSvc, interval: interval}
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
	analysis, err := j.reporter.AnalyzeScheduled(ctx, req, schedule.ID)
	if err != nil {
		slog.Error("scheduled log report failed",
			slog.String("schedule_id", schedule.ID.String()),
			slog.Any("error", err),
		)
		return
	}
	slog.Info("scheduled log report completed", slog.String("schedule_id", schedule.ID.String()))

	// Deliver the report to the configured notification channels. Previously
	// DeliveryChannelIDs was stored but never used, so reports were generated
	// but never actually sent to anyone.
	if j.notifSvc != nil && len(schedule.DeliveryChannelIDs) > 0 {
		subject := "Geplanter Log-Report"
		body := buildLogReportBody(schedule, analysis)
		attempted, delivered := j.notifSvc.NotifyChannels(ctx, schedule.DeliveryChannelIDs, "log_report", subject, body)
		slog.Info("log report delivered",
			slog.String("schedule_id", schedule.ID.String()),
			slog.Int("attempted", attempted),
			slog.Int("delivered", delivered),
		)
	}
}

// buildLogReportBody renders a short text body for the report delivery from the
// analysis's stored report JSON.
func buildLogReportBody(schedule model.LogReportSchedule, analysis *model.LogAnalysis) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Log-Report für %d Node(s), Zeitfenster %dh.\n", len(schedule.NodeIDs), schedule.TimeWindowHours)
	if analysis != nil {
		var report model.LogAnalysisReport
		if len(analysis.ReportJSON) > 0 && json.Unmarshal(analysis.ReportJSON, &report) == nil {
			if report.Summary != "" {
				fmt.Fprintf(&sb, "\nZusammenfassung:\n%s\n", report.Summary)
			}
			if len(report.Anomalies) > 0 {
				fmt.Fprintf(&sb, "\nAnomalien: %d\n", len(report.Anomalies))
			}
			if len(report.Recommendations) > 0 {
				sb.WriteString("\nEmpfehlungen:\n")
				for _, r := range report.Recommendations {
					fmt.Fprintf(&sb, "  - %s\n", r)
				}
			}
		}
	}
	return sb.String()
}
