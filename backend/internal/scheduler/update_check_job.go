package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/updates"
)

type UpdateCheckJob struct {
	updateSvc *updates.Service
	nodeRepo  repository.NodeRepository
	interval  time.Duration
}

func NewUpdateCheckJob(updateSvc *updates.Service, nodeRepo repository.NodeRepository, interval time.Duration) *UpdateCheckJob {
	return &UpdateCheckJob{
		updateSvc: updateSvc,
		nodeRepo:  nodeRepo,
		interval:  interval,
	}
}

func (j *UpdateCheckJob) Name() string {
	return "update_check"
}

func (j *UpdateCheckJob) Interval() time.Duration {
	return j.interval
}

func (j *UpdateCheckJob) Run(ctx context.Context) error {
	nodes, err := j.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	checked, failed := 0, 0

	for _, n := range nodes {
		if !n.IsOnline {
			continue
		}
		node := n
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := j.updateSvc.CheckUpdates(ctx, node.ID); err != nil {
				slog.Warn("update check failed",
					slog.String("node", node.Name),
					slog.Any("error", err),
				)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}
			mu.Lock()
			checked++
			mu.Unlock()
		}()
	}

	wg.Wait()
	slog.Info("update check job completed",
		slog.Int("checked", checked),
		slog.Int("failed", failed),
	)
	return nil
}
