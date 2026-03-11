package vm

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type HealthService struct {
	nodeRepo      repository.NodeRepository
	metricsRepo   repository.MetricsRepository
	clientFactory proxmox.ClientFactory
}

func NewHealthService(
	nodeRepo repository.NodeRepository,
	metricsRepo repository.MetricsRepository,
	clientFactory proxmox.ClientFactory,
) *HealthService {
	return &HealthService{
		nodeRepo:      nodeRepo,
		metricsRepo:   metricsRepo,
		clientFactory: clientFactory,
	}
}

// CalculateHealthScore computes a 0-100 health score for a VM.
// Weights: CPU 25%, RAM 25%, Disk 25%, Stability 25%.
func (s *HealthService) CalculateHealthScore(ctx context.Context, nodeID uuid.UUID, vmid int) (*model.HealthScore, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return nil, fmt.Errorf("create proxmox client: %w", err)
	}

	pveNodes, err := client.GetNodes(ctx)
	if err != nil || len(pveNodes) == 0 {
		return nil, fmt.Errorf("get pve nodes: %w", err)
	}

	// Get current VM info
	vms, err := client.GetVMs(ctx, pveNodes[0])
	if err != nil {
		return nil, fmt.Errorf("get vms: %w", err)
	}

	var vm *proxmox.VMInfo
	for _, v := range vms {
		if v.VMID == vmid {
			vm = &v
			break
		}
	}
	if vm == nil {
		return nil, fmt.Errorf("vm %d not found", vmid)
	}

	// Get 7-day RRD data
	rrdData, err := client.GetVMRRDData(ctx, pveNodes[0], vmid, vm.Type, "week")
	if err != nil {
		rrdData = nil // proceed with current data only
	}

	breakdown := model.HealthBreakdown{}

	// CPU Score (25 points)
	// Good: avg < 80%, Warning: 80-95%, Critical: >95%
	var cpuAvg float64
	if len(rrdData) > 0 {
		var total float64
		for _, dp := range rrdData {
			total += dp.CPU
		}
		cpuAvg = (total / float64(len(rrdData))) * 100
	} else {
		cpuAvg = vm.CPU * 100
	}
	breakdown.CPUAvg = cpuAvg
	if cpuAvg < 70 {
		breakdown.CPUScore = 25
	} else if cpuAvg < 85 {
		breakdown.CPUScore = 20
	} else if cpuAvg < 95 {
		breakdown.CPUScore = 10
	} else {
		breakdown.CPUScore = 5
	}

	// RAM Score (25 points)
	var ramAvg float64
	if len(rrdData) > 0 {
		var total float64
		count := 0
		for _, dp := range rrdData {
			if dp.MaxMem > 0 {
				total += (dp.Mem / dp.MaxMem) * 100
				count++
			}
		}
		if count > 0 {
			ramAvg = total / float64(count)
		}
	} else if vm.MaxMem > 0 {
		ramAvg = float64(vm.Mem) / float64(vm.MaxMem) * 100
	}
	breakdown.RAMAvg = ramAvg
	if ramAvg < 70 {
		breakdown.RAMScore = 25
	} else if ramAvg < 85 {
		breakdown.RAMScore = 20
	} else if ramAvg < 95 {
		breakdown.RAMScore = 10
	} else {
		breakdown.RAMScore = 5
	}

	// Disk Score (25 points)
	var diskUsage float64
	if vm.MaxDisk > 0 {
		diskUsage = float64(vm.Disk) / float64(vm.MaxDisk) * 100
	}
	breakdown.DiskUsage = diskUsage
	if diskUsage < 70 {
		breakdown.DiskScore = 25
	} else if diskUsage < 85 {
		breakdown.DiskScore = 20
	} else if diskUsage < 95 {
		breakdown.DiskScore = 10
	} else {
		breakdown.DiskScore = 5
	}

	// Stability Score (25 points)
	// Based on uptime: >7d = 25, >1d = 20, >1h = 10, else 5
	uptimeDays := float64(vm.Uptime) / 86400.0
	breakdown.UptimeDays = math.Round(uptimeDays*10) / 10

	// Estimate crash count from VM metrics: count gaps > 10 min in last 7 days
	crashCount := 0
	end := time.Now()
	start := end.Add(-7 * 24 * time.Hour)
	vmMetrics, _ := s.metricsRepo.GetVMMetricsHistory(ctx, nodeID, vmid, start, end)
	if len(vmMetrics) > 1 {
		for i := 1; i < len(vmMetrics); i++ {
			gap := vmMetrics[i].RecordedAt.Sub(vmMetrics[i-1].RecordedAt)
			if gap > 10*time.Minute {
				crashCount++
			}
		}
	}
	breakdown.CrashCount = crashCount

	if uptimeDays >= 7 && crashCount == 0 {
		breakdown.StabilityScore = 25
	} else if uptimeDays >= 1 && crashCount <= 1 {
		breakdown.StabilityScore = 20
	} else if uptimeDays >= 0.04 { // ~1 hour
		breakdown.StabilityScore = 10
	} else {
		breakdown.StabilityScore = 5
	}

	score := breakdown.CPUScore + breakdown.RAMScore + breakdown.DiskScore + breakdown.StabilityScore

	status := "healthy"
	if score <= 50 {
		status = "critical"
	} else if score <= 80 {
		status = "warning"
	}

	return &model.HealthScore{
		NodeID:    nodeID,
		VMID:      vmid,
		VMName:    vm.Name,
		VMType:    vm.Type,
		Score:     score,
		Status:    status,
		Breakdown: breakdown,
		UpdatedAt: time.Now(),
	}, nil
}

// CalculateAllHealthScores computes health scores for all running VMs on a node.
func (s *HealthService) CalculateAllHealthScores(ctx context.Context, nodeID uuid.UUID) ([]model.HealthScore, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	client, err := s.clientFactory.CreateClient(node)
	if err != nil {
		return nil, fmt.Errorf("create proxmox client: %w", err)
	}

	pveNodes, err := client.GetNodes(ctx)
	if err != nil || len(pveNodes) == 0 {
		return nil, fmt.Errorf("get pve nodes: %w", err)
	}

	vms, err := client.GetVMs(ctx, pveNodes[0])
	if err != nil {
		return nil, fmt.Errorf("get vms: %w", err)
	}

	var scores []model.HealthScore
	for _, vm := range vms {
		if vm.Status != "running" {
			continue
		}
		score, err := s.CalculateHealthScore(ctx, nodeID, vm.VMID)
		if err != nil {
			continue
		}
		scores = append(scores, *score)
	}

	return scores, nil
}
