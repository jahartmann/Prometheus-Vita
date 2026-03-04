package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/recovery"
)

type DRProfileJob struct {
	nodeRepo     repository.NodeRepository
	profileSvc   *recovery.ProfileService
	readinessSvc *recovery.ReadinessService
	interval     time.Duration
}

func NewDRProfileJob(
	nodeRepo repository.NodeRepository,
	profileSvc *recovery.ProfileService,
	readinessSvc *recovery.ReadinessService,
	interval time.Duration,
) *DRProfileJob {
	return &DRProfileJob{
		nodeRepo:     nodeRepo,
		profileSvc:   profileSvc,
		readinessSvc: readinessSvc,
		interval:     interval,
	}
}

func (j *DRProfileJob) Name() string {
	return "dr_profile_collection"
}

func (j *DRProfileJob) Interval() time.Duration {
	return j.interval
}

func (j *DRProfileJob) Run(ctx context.Context) error {
	nodes, err := j.nodeRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if !node.IsOnline {
			continue
		}

		// Collect profile
		if _, err := j.profileSvc.CollectProfile(ctx, node.ID); err != nil {
			slog.Error("failed to collect DR profile",
				slog.String("node_id", node.ID.String()),
				slog.String("node_name", node.Name),
				slog.Any("error", err),
			)
			continue
		}

		// Calculate readiness score
		if _, err := j.readinessSvc.CalculateScore(ctx, node.ID); err != nil {
			slog.Error("failed to calculate DR readiness score",
				slog.String("node_id", node.ID.String()),
				slog.String("node_name", node.Name),
				slog.Any("error", err),
			)
		}
	}

	return nil
}
