package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/rightsizing"
)

type RightsizingJob struct {
	rightsizingSvc *rightsizing.Service
	nodeRepo       repository.NodeRepository
	interval       time.Duration
}

func NewRightsizingJob(rightsizingSvc *rightsizing.Service, nodeRepo repository.NodeRepository, interval time.Duration) *RightsizingJob {
	return &RightsizingJob{
		rightsizingSvc: rightsizingSvc,
		nodeRepo:       nodeRepo,
		interval:       interval,
	}
}

func (j *RightsizingJob) Name() string {
	return "rightsizing"
}

func (j *RightsizingJob) Interval() time.Duration {
	return j.interval
}

func (j *RightsizingJob) Run(ctx context.Context) error {
	nodes, err := j.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	analyzed, failed := 0, 0

	for _, n := range nodes {
		if !n.IsOnline {
			continue
		}
		node := n
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := j.rightsizingSvc.AnalyzeNode(ctx, node.ID); err != nil {
				slog.Warn("rightsizing analysis failed",
					slog.String("node", node.Name),
					slog.Any("error", err),
				)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}
			mu.Lock()
			analyzed++
			mu.Unlock()
		}()
	}

	wg.Wait()
	slog.Info("rightsizing job completed",
		slog.Int("analyzed", analyzed),
		slog.Int("failed", failed),
	)
	return nil
}
