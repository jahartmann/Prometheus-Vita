package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/vm"
)

// ScheduledActionJob evaluates active scheduled VM actions (start/stop/shutdown/
// restart) and executes the ones whose cron schedule is due. Without this job
// the actions were stored and shown as "active" but never fired.
type ScheduledActionJob struct {
	actionRepo repository.ScheduledActionRepository
	actionSvc  *vm.ScheduledActionService
	interval   time.Duration
}

func NewScheduledActionJob(
	actionRepo repository.ScheduledActionRepository,
	actionSvc *vm.ScheduledActionService,
	interval time.Duration,
) *ScheduledActionJob {
	return &ScheduledActionJob{actionRepo: actionRepo, actionSvc: actionSvc, interval: interval}
}

func (j *ScheduledActionJob) Name() string            { return "scheduled_action" }
func (j *ScheduledActionJob) Interval() time.Duration { return j.interval }

func (j *ScheduledActionJob) Run(ctx context.Context) error {
	actions, err := j.actionRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active scheduled actions: %w", err)
	}

	now := time.Now()
	for _, a := range actions {
		if !isDue(a.ScheduleCron, a.LastRunAt, a.CreatedAt, now) {
			continue
		}
		slog.Info("executing due scheduled action",
			slog.String("action_id", a.ID.String()),
			slog.String("action", a.Action),
		)
		if err := j.actionSvc.Execute(ctx, a); err != nil {
			slog.Error("scheduled action execution failed",
				slog.String("action_id", a.ID.String()),
				slog.Any("error", err),
			)
			continue
		}
		// Record the run only on success so a transient failure is retried on
		// the next tick rather than silently skipped for the whole interval.
		if err := j.actionRepo.UpdateLastRun(ctx, a.ID, now); err != nil {
			slog.Warn("failed to update scheduled action last_run",
				slog.String("action_id", a.ID.String()),
				slog.Any("error", err),
			)
		}
	}
	return nil
}
