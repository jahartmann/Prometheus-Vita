package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/service/briefing"
)

type BriefingJob struct {
	briefingSvc *briefing.Service
	interval    time.Duration
	hour        int
	lastRunDate string
}

func NewBriefingJob(briefingSvc *briefing.Service, hour int) *BriefingJob {
	return &BriefingJob{
		briefingSvc: briefingSvc,
		interval:    15 * time.Minute, // Check every 15 minutes
		hour:        hour,
	}
}

func (j *BriefingJob) Name() string {
	return "morning_briefing"
}

func (j *BriefingJob) Interval() time.Duration {
	return j.interval
}

func (j *BriefingJob) Run(ctx context.Context) error {
	now := time.Now()
	today := now.Format("2006-01-02")

	// Only run once per day at the configured hour
	if now.Hour() != j.hour || j.lastRunDate == today {
		return nil
	}

	// Check DB to prevent duplicates after restart
	existing, err := j.briefingSvc.GetLatest(ctx)
	if err == nil && existing != nil && existing.GeneratedAt.Format("2006-01-02") == today {
		j.lastRunDate = today
		return nil
	}

	j.lastRunDate = today
	slog.Info("generating morning briefing", slog.Int("hour", j.hour))

	return j.briefingSvc.GenerateBriefing(ctx)
}
