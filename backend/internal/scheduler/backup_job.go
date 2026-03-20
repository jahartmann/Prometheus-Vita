package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/backup"
)

// BackupScheduleJob is a scheduler Job that checks for due backup schedules
// and triggers automated backups. After each backup it enforces the
// schedule's retention policy by deleting the oldest backups beyond the
// configured limit.
type BackupScheduleJob struct {
	scheduleRepo repository.ScheduleRepository
	backupRepo   repository.BackupRepository
	backupSvc    *backup.Service
	interval     time.Duration
}

// NewBackupScheduleJob creates a new BackupScheduleJob that polls for due
// schedules at the given interval.
func NewBackupScheduleJob(
	scheduleRepo repository.ScheduleRepository,
	backupRepo repository.BackupRepository,
	backupSvc *backup.Service,
	interval time.Duration,
) *BackupScheduleJob {
	return &BackupScheduleJob{
		scheduleRepo: scheduleRepo,
		backupRepo:   backupRepo,
		backupSvc:    backupSvc,
		interval:     interval,
	}
}

// Name returns the unique name of this job for logging purposes.
func (j *BackupScheduleJob) Name() string {
	return "backup_schedule"
}

// Interval returns how often the scheduler should invoke this job.
func (j *BackupScheduleJob) Interval() time.Duration {
	return j.interval
}

// Run executes one pass of the backup schedule job. It queries for all
// schedules whose next_run_at is in the past, creates a backup for each,
// advances each schedule's next run time, and enforces retention limits.
func (j *BackupScheduleJob) Run(ctx context.Context) error {
	schedules, err := j.scheduleRepo.ListDue(ctx)
	if err != nil {
		return fmt.Errorf("list due schedules: %w", err)
	}

	if len(schedules) == 0 {
		return nil
	}

	slog.Info("processing due backup schedules", slog.Int("count", len(schedules)))

	for _, schedule := range schedules {
		j.processSchedule(ctx, schedule)
	}

	return nil
}

// processSchedule handles a single due schedule: creates the backup,
// calculates the next run time, and enforces retention.
func (j *BackupScheduleJob) processSchedule(ctx context.Context, schedule model.BackupSchedule) {
	slog.Info("executing scheduled backup",
		slog.String("schedule_id", schedule.ID.String()),
		slog.String("node_id", schedule.NodeID.String()),
	)

	// Calculate next run time from the scheduled time (not time.Now()) to
	// prevent schedule drift when backups take a long time.
	now := time.Now()
	nextRun, err := backup.NextRun(schedule.CronExpression, schedule.NextRunAt)
	if err != nil {
		slog.Error("failed to calculate next run time",
			slog.String("schedule_id", schedule.ID.String()),
			slog.String("cron", schedule.CronExpression),
			slog.Any("error", err),
		)
		return
	}

	// Advance next_run_at BEFORE creating the backup so that a slow backup
	// does not cause duplicate triggers on the next scheduler tick.
	if err := j.scheduleRepo.UpdateNextRun(ctx, schedule.ID, now, nextRun); err != nil {
		slog.Error("failed to update schedule next run",
			slog.String("schedule_id", schedule.ID.String()),
			slog.Any("error", err),
		)
	}

	// Create the backup
	req := model.CreateBackupRequest{
		BackupType: model.BackupTypeScheduled,
		Notes:      fmt.Sprintf("Automated backup from schedule %s", schedule.ID.String()),
	}

	_, err = j.backupSvc.CreateBackup(ctx, schedule.NodeID, req)
	if err != nil {
		slog.Error("scheduled backup failed",
			slog.String("schedule_id", schedule.ID.String()),
			slog.String("node_id", schedule.NodeID.String()),
			slog.Any("error", err),
		)
	}

	// Enforce retention policy
	if schedule.RetentionCount > 0 {
		count, err := j.backupRepo.CountByNode(ctx, schedule.NodeID)
		if err != nil {
			slog.Error("failed to count backups for retention",
				slog.String("node_id", schedule.NodeID.String()),
				slog.Any("error", err),
			)
			return
		}

		if count > schedule.RetentionCount {
			if err := j.backupRepo.DeleteOldest(ctx, schedule.NodeID, schedule.RetentionCount); err != nil {
				slog.Error("failed to delete oldest backups for retention",
					slog.String("node_id", schedule.NodeID.String()),
					slog.Any("error", err),
				)
			} else {
				slog.Info("enforced backup retention policy",
					slog.String("node_id", schedule.NodeID.String()),
					slog.Int("retention_count", schedule.RetentionCount),
					slog.Int("total_before", count),
				)
			}
		}
	}
}
