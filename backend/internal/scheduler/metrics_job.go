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
	"github.com/google/uuid"
)

// previousReading stores the last cumulative counter values for delta calculation.
type previousReading struct {
	netIn     int64
	netOut    int64
	diskRead  int64
	diskWrite int64
	readAt    time.Time
}

// metricsCache stores previous readings keyed by identifier (nodeID or nodeID:vmid).
type metricsCache struct {
	mu       sync.Mutex
	previous map[string]*previousReading
}

func newMetricsCache() *metricsCache {
	return &metricsCache{
		previous: make(map[string]*previousReading),
	}
}

// calculateDelta returns per-second rates from cumulative counters.
// Returns (netInRate, netOutRate, diskReadRate, diskWriteRate, valid).
// If there's no previous reading or the counter reset, returns 0 values.
func (mc *metricsCache) calculateDelta(key string, netIn, netOut, diskRead, diskWrite int64, now time.Time) (int64, int64, int64, int64, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	prev, exists := mc.previous[key]

	// Always update with current reading
	mc.previous[key] = &previousReading{
		netIn:     netIn,
		netOut:    netOut,
		diskRead:  diskRead,
		diskWrite: diskWrite,
		readAt:    now,
	}

	if !exists || prev == nil {
		return 0, 0, 0, 0, false
	}

	elapsed := now.Sub(prev.readAt).Seconds()
	if elapsed <= 0 {
		return 0, 0, 0, 0, false
	}

	// Calculate deltas; if counter reset (new < old), treat as 0
	calcRate := func(current, previous int64) int64 {
		delta := current - previous
		if delta < 0 {
			// Counter reset (reboot, overflow); return 0 for this interval
			return 0
		}
		rate := float64(delta) / elapsed
		return int64(rate)
	}

	netInRate := calcRate(netIn, prev.netIn)
	netOutRate := calcRate(netOut, prev.netOut)
	diskReadRate := calcRate(diskRead, prev.diskRead)
	diskWriteRate := calcRate(diskWrite, prev.diskWrite)

	return netInRate, netOutRate, diskReadRate, diskWriteRate, true
}

type MetricsCollectionJob struct {
	nodeRepo      repository.NodeRepository
	metricsRepo   repository.MetricsRepository
	clientFactory proxmox.ClientFactory
	wsHub         *monitor.WSHub
	interval      time.Duration
	cache         *metricsCache
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
		cache:         newMetricsCache(),
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

			pveNode := nodeService.ResolvePVENode(&node, pveNodes)
			status, err := client.GetNodeStatus(ctx, pveNode)
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

			now := time.Now().UTC()

			// Calculate network rate from cumulative counters
			nodeKey := node.ID.String()
			netInRate, netOutRate, _, _, valid := j.cache.calculateDelta(
				nodeKey, status.NetIn, status.NetOut, 0, 0, now,
			)

			// If first reading (no previous), store 0 rates
			if !valid {
				netInRate = 0
				netOutRate = 0
			}

			record := &model.MetricsRecord{
				NodeID:     node.ID,
				RecordedAt: now,
				CPUUsage:   status.CPUUsage,
				MemUsed:    status.MemUsed,
				MemTotal:   status.MemTotal,
				DiskUsed:   status.DiskUsed,
				DiskTotal:  status.DiskTotal,
				NetIn:      netInRate,
				NetOut:     netOutRate,
				LoadAvg:    status.LoadAvg,
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

			// Collect per-VM metrics
			j.collectVMMetrics(ctx, client, pveNode, node.ID, now)

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

	// Cleanup VM metrics older than 7 days
	vmDeleted, err := j.metricsRepo.CleanupVMMetrics(ctx, cutoff)
	if err != nil {
		slog.Warn("metrics: failed to cleanup old vm metrics", slog.Any("error", err))
	}

	slog.Info("metrics collection completed",
		slog.Int("collected", collected),
		slog.Int("failed", failed),
		slog.Int64("cleaned_up", deleted),
		slog.Int64("vm_cleaned_up", vmDeleted),
	)

	return nil
}

// collectVMMetrics fetches per-VM metrics and stores them with delta-calculated network rates.
func (j *MetricsCollectionJob) collectVMMetrics(ctx context.Context, client *proxmox.Client, pveNode string, nodeID uuid.UUID, now time.Time) {
	vms, err := client.GetVMs(ctx, pveNode)
	if err != nil {
		slog.Warn("metrics: failed to get VMs for VM metrics",
			slog.String("node_id", nodeID.String()),
			slog.Any("error", err),
		)
		return
	}

	for _, vm := range vms {
		if vm.Status != "running" {
			continue
		}

		vmKey := fmt.Sprintf("%s:%d", nodeID.String(), vm.VMID)
		netInRate, netOutRate, diskReadRate, diskWriteRate, valid := j.cache.calculateDelta(
			vmKey, int64(vm.NetIn), int64(vm.NetOut), int64(vm.DiskRead), int64(vm.DiskWrite), now,
		)

		if !valid {
			netInRate = 0
			netOutRate = 0
			diskReadRate = 0
			diskWriteRate = 0
		}

		vmRecord := &model.VMMetricsRecord{
			NodeID:     nodeID,
			VMID:       vm.VMID,
			VMType:     vm.Type,
			CPUUsage:   vm.CPU * 100,
			MemUsed:    vm.Mem,
			MemTotal:   vm.MaxMem,
			NetIn:      netInRate,
			NetOut:     netOutRate,
			DiskRead:   diskReadRate,
			DiskWrite:  diskWriteRate,
			RecordedAt: now,
		}

		if err := j.metricsRepo.InsertVMMetrics(ctx, vmRecord); err != nil {
			slog.Warn("metrics: failed to insert vm metrics",
				slog.Int("vmid", vm.VMID),
				slog.Any("error", err),
			)
		}

		// Broadcast VM metrics via WebSocket for live frontend updates
		j.wsHub.BroadcastMessage(monitor.WSMessage{
			Type: "vm_metrics",
			Data: vmRecord,
		})
	}
}
