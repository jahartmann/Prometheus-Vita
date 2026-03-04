package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	nodeService "github.com/antigravity/prometheus/internal/service/node"
	"github.com/antigravity/prometheus/internal/service/monitor"
)

type MetricsCollectionJob struct {
	nodeRepo      repository.NodeRepository
	metricsRepo   repository.MetricsRepository
	clientFactory proxmox.ClientFactory
	wsHub         *monitor.WSHub
	interval      time.Duration
}

func NewMetricsCollectionJob(
	nodeRepo repository.NodeRepository,
	metricsRepo repository.MetricsRepository,
	clientFactory proxmox.ClientFactory,
	wsHub *monitor.WSHub,
	interval time.Duration,
) *MetricsCollectionJob {
	return &MetricsCollectionJob{
		nodeRepo:      nodeRepo,
		metricsRepo:   metricsRepo,
		clientFactory: clientFactory,
		wsHub:         wsHub,
		interval:      interval,
	}
}

func (j *MetricsCollectionJob) Name() string {
	return "metrics_collection"
}

func (j *MetricsCollectionJob) Interval() time.Duration {
	return j.interval
}

func (j *MetricsCollectionJob) Run(ctx context.Context) error {
	nodes, err := j.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	collected := 0
	failed := 0

	for _, n := range nodes {
		if !n.IsOnline {
			continue
		}

		node := n
		wg.Add(1)
		go func() {
			defer wg.Done()

			client, err := j.clientFactory.CreateClient(&node)
			if err != nil {
				slog.Warn("metrics: failed to create proxmox client",
					slog.String("node", node.Name),
					slog.Any("error", err),
				)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			pveNodes, err := client.GetNodes(ctx)
			if err != nil || len(pveNodes) == 0 {
				slog.Warn("metrics: failed to get pve nodes",
					slog.String("node", node.Name),
					slog.Any("error", err),
				)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			status, err := client.GetNodeStatus(ctx, nodeService.ResolvePVENode(&node, pveNodes))
			if err != nil {
				slog.Warn("metrics: failed to get node status",
					slog.String("node", node.Name),
					slog.Any("error", err),
				)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			record := &model.MetricsRecord{
				NodeID:     node.ID,
				RecordedAt: time.Now().UTC(),
				CPUUsage:   status.CPUUsage,
				MemUsed:    status.MemUsed,
				MemTotal:   status.MemTotal,
				DiskUsed:   status.DiskUsed,
				DiskTotal:  status.DiskTotal,
				NetIn:      0,
				NetOut:     0,
				LoadAvg:    []float64{},
			}

			if err := j.metricsRepo.Insert(ctx, record); err != nil {
				slog.Warn("metrics: failed to insert record",
					slog.String("node", node.Name),
					slog.Any("error", err),
				)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			j.wsHub.BroadcastMessage(monitor.WSMessage{
				Type: "node_metrics",
				Data: record,
			})

			mu.Lock()
			collected++
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Cleanup records older than 7 days
	cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour)
	deleted, err := j.metricsRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		slog.Warn("metrics: failed to cleanup old records", slog.Any("error", err))
	}

	slog.Info("metrics collection completed",
		slog.Int("collected", collected),
		slog.Int("failed", failed),
		slog.Int64("cleaned_up", deleted),
	)

	return nil
}
