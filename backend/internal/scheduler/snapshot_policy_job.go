package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/vm"
)

// SnapshotPolicyJob evaluates active VM snapshot policies and executes the ones
// whose cron schedule is due. Without this job the policies were stored and
// shown as "active" but never actually created any snapshots.
type SnapshotPolicyJob struct {
	policyRepo repository.SnapshotPolicyRepository
	policySvc  *vm.SnapshotPolicyService
	interval   time.Duration
}

func NewSnapshotPolicyJob(
	policyRepo repository.SnapshotPolicyRepository,
	policySvc *vm.SnapshotPolicyService,
	interval time.Duration,
) *SnapshotPolicyJob {
	return &SnapshotPolicyJob{policyRepo: policyRepo, policySvc: policySvc, interval: interval}
}

func (j *SnapshotPolicyJob) Name() string            { return "snapshot_policy" }
func (j *SnapshotPolicyJob) Interval() time.Duration { return j.interval }

func (j *SnapshotPolicyJob) Run(ctx context.Context) error {
	policies, err := j.policyRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active snapshot policies: %w", err)
	}

	now := time.Now()
	for _, p := range policies {
		if !isDue(p.ScheduleCron, p.LastRun, p.CreatedAt, now) {
			continue
		}
		policy := p // avoid capturing the loop variable by reference
		slog.Info("executing due snapshot policy",
			slog.String("policy_id", policy.ID.String()),
			slog.Int("vmid", policy.VMID),
		)
		// ExecutePolicy creates the snapshot, enforces retention and updates
		// last_run, so a successful run advances the schedule.
		if err := j.policySvc.ExecutePolicy(ctx, &policy); err != nil {
			slog.Error("snapshot policy execution failed",
				slog.String("policy_id", policy.ID.String()),
				slog.Any("error", err),
			)
		}
	}
	return nil
}
