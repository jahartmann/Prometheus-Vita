package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	nodeRepo    repository.NodeRepository
	metricsRepo repository.MetricsRepository
	redis       *redis.Client
}

func NewService(nodeRepo repository.NodeRepository, redisClient *redis.Client, metricsRepo repository.MetricsRepository) *Service {
	return &Service{
		nodeRepo:    nodeRepo,
		metricsRepo: metricsRepo,
		redis:       redisClient,
	}
}

func (s *Service) GetCachedStatus(ctx context.Context, nodeID uuid.UUID) (*proxmox.NodeStatus, error) {
	key := fmt.Sprintf("node:status:%s", nodeID.String())

	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get cached status: %w", err)
	}

	var status proxmox.NodeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal cached status: %w", err)
	}

	return &status, nil
}

type NodeStatusSummary struct {
	NodeID   uuid.UUID             `json:"node_id"`
	NodeName string                `json:"node_name"`
	IsOnline bool                  `json:"is_online"`
	Status   *proxmox.NodeStatus   `json:"status,omitempty"`
}

func (s *Service) GetAllNodesStatus(ctx context.Context) ([]NodeStatusSummary, error) {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	summaries := make([]NodeStatusSummary, 0, len(nodes))
	for _, node := range nodes {
		summary := NodeStatusSummary{
			NodeID:   node.ID,
			NodeName: node.Name,
			IsOnline: node.IsOnline,
		}

		if node.IsOnline {
			status, _ := s.GetCachedStatus(ctx, node.ID)
			summary.Status = status
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (s *Service) GetMetricsHistory(ctx context.Context, nodeID uuid.UUID, since, until time.Time) ([]model.MetricsRecord, error) {
	records, err := s.metricsRepo.GetByNode(ctx, nodeID, since, until)
	if err != nil {
		return nil, fmt.Errorf("get metrics history: %w", err)
	}
	if records == nil {
		records = []model.MetricsRecord{}
	}
	return records, nil
}

func (s *Service) GetMetricsSummary(ctx context.Context, nodeID uuid.UUID, since, until time.Time) (*model.MetricsSummary, error) {
	records, err := s.metricsRepo.GetByNode(ctx, nodeID, since, until)
	if err != nil {
		return nil, fmt.Errorf("get metrics for summary: %w", err)
	}

	summary := &model.MetricsSummary{
		NodeID: nodeID,
	}

	if len(records) == 0 {
		return summary, nil
	}

	var cpuSum, memPctSum, diskPctSum float64
	cpuMin := math.MaxFloat64
	cpuMax := -math.MaxFloat64
	memPctMin := math.MaxFloat64
	memPctMax := -math.MaxFloat64
	diskPctMin := math.MaxFloat64
	diskPctMax := -math.MaxFloat64

	for _, r := range records {
		cpuSum += r.CPUUsage
		if r.CPUUsage < cpuMin {
			cpuMin = r.CPUUsage
		}
		if r.CPUUsage > cpuMax {
			cpuMax = r.CPUUsage
		}

		var memPct float64
		if r.MemTotal > 0 {
			memPct = float64(r.MemUsed) / float64(r.MemTotal) * 100
		}
		memPctSum += memPct
		if memPct > memPctMax {
			memPctMax = memPct
		}
		if memPct < memPctMin {
			memPctMin = memPct
		}

		var diskPct float64
		if r.DiskTotal > 0 {
			diskPct = float64(r.DiskUsed) / float64(r.DiskTotal) * 100
		}
		diskPctSum += diskPct
		if diskPct > diskPctMax {
			diskPctMax = diskPct
		}
		if diskPct < diskPctMin {
			diskPctMin = diskPct
		}
	}

	n := float64(len(records))
	summary.CPUAvg = cpuSum / n
	summary.CPUMax = cpuMax
	summary.CPUMin = cpuMin
	summary.MemAvg = memPctSum / n
	summary.MemMax = memPctMax
	summary.MemMin = memPctMin
	summary.DiskAvg = diskPctSum / n
	summary.DiskMax = diskPctMax
	summary.DiskMin = diskPctMin

	// Get current values from the latest record
	latest := records[len(records)-1]
	summary.CPUCurrent = latest.CPUUsage
	if latest.MemTotal > 0 {
		summary.MemCurrent = float64(latest.MemUsed) / float64(latest.MemTotal) * 100
	}
	if latest.DiskTotal > 0 {
		summary.DiskCurrent = float64(latest.DiskUsed) / float64(latest.DiskTotal) * 100
	}

	return summary, nil
}

type ClusterHistoryPoint struct {
	Time    time.Time `json:"time"`
	CPUAvg  float64   `json:"cpu_avg"`
	MemPct  float64   `json:"mem_pct"`
	DiskPct float64   `json:"disk_pct"`
	NetIn   int64     `json:"net_in"`
	NetOut  int64     `json:"net_out"`
}

func (s *Service) GetClusterHistory(ctx context.Context, since, until time.Time) ([]ClusterHistoryPoint, error) {
	records, err := s.metricsRepo.GetAllMetrics(ctx, since, until)
	if err != nil {
		return nil, fmt.Errorf("get cluster history: %w", err)
	}

	// Group by time bucket (5-min intervals)
	buckets := map[time.Time][]model.MetricsRecord{}
	for _, r := range records {
		bucket := r.RecordedAt.Truncate(5 * time.Minute)
		buckets[bucket] = append(buckets[bucket], r)
	}

	// Aggregate each bucket
	points := make([]ClusterHistoryPoint, 0, len(buckets))
	for t, recs := range buckets {
		var cpuSum, memUsed, memTotal, diskUsed, diskTotal float64
		var netIn, netOut int64
		for _, r := range recs {
			cpuSum += r.CPUUsage
			memUsed += float64(r.MemUsed)
			memTotal += float64(r.MemTotal)
			diskUsed += float64(r.DiskUsed)
			diskTotal += float64(r.DiskTotal)
			netIn += r.NetIn
			netOut += r.NetOut
		}
		n := float64(len(recs))
		memPct := 0.0
		if memTotal > 0 {
			memPct = memUsed / memTotal * 100
		}
		diskPct := 0.0
		if diskTotal > 0 {
			diskPct = diskUsed / diskTotal * 100
		}
		points = append(points, ClusterHistoryPoint{
			Time:    t,
			CPUAvg:  cpuSum / n,
			MemPct:  memPct,
			DiskPct: diskPct,
			NetIn:   netIn / int64(n),
			NetOut:  netOut / int64(n),
		})
	}

	// Sort by time
	for i := 0; i < len(points)-1; i++ {
		for j := i + 1; j < len(points); j++ {
			if points[j].Time.Before(points[i].Time) {
				points[i], points[j] = points[j], points[i]
			}
		}
	}

	return points, nil
}

func (s *Service) GetVMMetricsHistory(ctx context.Context, nodeID uuid.UUID, vmid int, start, end time.Time) ([]model.VMMetricsRecord, error) {
	records, err := s.metricsRepo.GetVMMetricsHistory(ctx, nodeID, vmid, start, end)
	if err != nil {
		return nil, fmt.Errorf("get vm metrics history: %w", err)
	}
	if records == nil {
		records = []model.VMMetricsRecord{}
	}
	return records, nil
}

func (s *Service) GetVMNetworkSummary(ctx context.Context, nodeID uuid.UUID, vmid int, start, end time.Time) (*model.NetworkSummary, error) {
	return s.metricsRepo.GetVMNetworkSummary(ctx, nodeID, vmid, start, end)
}

func (s *Service) GetNodeNetworkSummary(ctx context.Context, nodeID uuid.UUID, start, end time.Time) (*model.NetworkSummary, error) {
	return s.metricsRepo.GetNodeNetworkSummary(ctx, nodeID, start, end)
}

func (s *Service) GetClusterNetworkSummary(ctx context.Context, start, end time.Time) (*model.NetworkSummary, error) {
	return s.metricsRepo.GetClusterNetworkSummary(ctx, start, end)
}
