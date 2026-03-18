package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/logscan"
)

// LogDiscoveryJob runs log source discovery on all active nodes.
type LogDiscoveryJob struct {
	discoverySvc *logscan.DiscoveryService
	nodeRepo     repository.NodeRepository
	interval     time.Duration
}

// NewLogDiscoveryJob creates a new LogDiscoveryJob.
func NewLogDiscoveryJob(discoverySvc *logscan.DiscoveryService, nodeRepo repository.NodeRepository, interval time.Duration) *LogDiscoveryJob {
	return &LogDiscoveryJob{
		discoverySvc: discoverySvc,
		nodeRepo:     nodeRepo,
		interval:     interval,
	}
}

func (j *LogDiscoveryJob) Name() string {
	return "log_discovery"
}

func (j *LogDiscoveryJob) Interval() time.Duration {
	return j.interval
}

func (j *LogDiscoveryJob) Run(ctx context.Context) error {
	nodes, err := j.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("log_discovery: list nodes: %w", err)
	}

	for _, node := range nodes {
		if _, err := j.discoverySvc.DiscoverSources(ctx, node.ID); err != nil {
			slog.Error("log_discovery: discover sources failed",
				slog.String("node_id", node.ID.String()),
				slog.Any("error", err),
			)
		}
	}
	return nil
}
