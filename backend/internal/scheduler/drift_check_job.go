package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/drift"
)

type DriftCheckJob struct {
	driftSvc *drift.Service
	nodeRepo repository.NodeRepository
	interval time.Duration
}

func NewDriftCheckJob(driftSvc *drift.Service, nodeRepo repository.NodeRepository, interval time.Duration) *DriftCheckJob {
	return &DriftCheckJob{
		driftSvc: driftSvc,
		nodeRepo: nodeRepo,
		interval: interval,
	}
}

func (j *DriftCheckJob) Name() string {
	return "drift_check"
}

func (j *DriftCheckJob) Interval() time.Duration {
	return j.interval
}

func (j *DriftCheckJob) Run(ctx context.Context) error {
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
			if _, err := j.driftSvc.CheckDrift(ctx, node.ID); err != nil {
				slog.Warn("drift check failed",
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
	slog.Info("drift check job completed",
		slog.Int("checked", checked),
		slog.Int("failed", failed),
	)
	return nil
}
