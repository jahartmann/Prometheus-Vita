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

type AnomalyService struct {
	nodeRepo      repository.NodeRepository
	metricsRepo   repository.MetricsRepository
	clientFactory proxmox.ClientFactory
}

func NewAnomalyService(
	nodeRepo repository.NodeRepository,
	metricsRepo repository.MetricsRepository,
	clientFactory proxmox.ClientFactory,
) *AnomalyService {
	return &AnomalyService{
		nodeRepo:      nodeRepo,
		metricsRepo:   metricsRepo,
		clientFactory: clientFactory,
	}
}

// DetectAnomalies checks a VM for anomalies using a 7-day baseline (mean + 2*stddev).
func (s *AnomalyService) DetectAnomalies(ctx context.Context, nodeID uuid.UUID, vmid int) ([]model.VMAnomaly, error) {
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

	// Get 7-day RRD data as baseline
	rrdData, err := client.GetVMRRDData(ctx, pveNodes[0], vmid, vm.Type, "week")
	if err != nil || len(rrdData) < 10 {
		return []model.VMAnomaly{}, nil // not enough data for baseline
	}

	var anomalies []model.VMAnomaly
	now := time.Now()

	// Use all-but-last as baseline, last as current
	baselineData := rrdData[:len(rrdData)-1]
	current := rrdData[len(rrdData)-1]

	// Check CPU
	cpuAnomaly := checkMetricAnomaly(baselineData, current, "cpu", vm, nodeID, now,
		func(dp proxmox.VMRRDDataPoint) float64 { return dp.CPU * 100 },
	)
	if cpuAnomaly != nil {
		anomalies = append(anomalies, *cpuAnomaly)
	}

	// Check RAM
	ramAnomaly := checkMetricAnomaly(baselineData, current, "memory", vm, nodeID, now,
		func(dp proxmox.VMRRDDataPoint) float64 {
			if dp.MaxMem > 0 {
				return (dp.Mem / dp.MaxMem) * 100
			}
			return 0
		},
	)
	if ramAnomaly != nil {
		anomalies = append(anomalies, *ramAnomaly)
	}

	// Check Network In
	netInAnomaly := checkMetricAnomaly(baselineData, current, "network_in", vm, nodeID, now,
		func(dp proxmox.VMRRDDataPoint) float64 { return dp.NetIn },
	)
	if netInAnomaly != nil {
		anomalies = append(anomalies, *netInAnomaly)
	}

	// Check Network Out
	netOutAnomaly := checkMetricAnomaly(baselineData, current, "network_out", vm, nodeID, now,
		func(dp proxmox.VMRRDDataPoint) float64 { return dp.NetOut },
	)
	if netOutAnomaly != nil {
		anomalies = append(anomalies, *netOutAnomaly)
	}

	return anomalies, nil
}

func checkMetricAnomaly(
	baseline []proxmox.VMRRDDataPoint,
	current proxmox.VMRRDDataPoint,
	metricName string,
	vm *proxmox.VMInfo,
	nodeID uuid.UUID,
	now time.Time,
	extract func(proxmox.VMRRDDataPoint) float64,
) *model.VMAnomaly {
	if len(baseline) < 5 {
		return nil
	}

	// Calculate mean and stddev
	var sum, sumSq float64
	for _, dp := range baseline {
		v := extract(dp)
		sum += v
		sumSq += v * v
	}
	n := float64(len(baseline))
	mean := sum / n
	variance := (sumSq / n) - (mean * mean)
	if variance < 0 {
		variance = 0
	}
	stddev := math.Sqrt(variance)

	currentVal := extract(current)

	// Avoid false positives when stddev is very small
	if stddev < 0.01 {
		stddev = 0.01
	}

	zScore := (currentVal - mean) / stddev

	// Threshold: 2 stddev
	if math.Abs(zScore) < 2.0 {
		return nil
	}

	severity := "warning"
	if math.Abs(zScore) >= 3.0 {
		severity = "critical"
	}

	metricLabels := map[string]string{
		"cpu":         "CPU",
		"memory":      "RAM",
		"network_in":  "Netzwerk (eingehend)",
		"network_out": "Netzwerk (ausgehend)",
	}
	label := metricLabels[metricName]
	if label == "" {
		label = metricName
	}

	direction := "ueber"
	if zScore < 0 {
		direction = "unter"
	}

	message := fmt.Sprintf("%s-Wert (%.1f) liegt %.1f Standardabweichungen %s dem Durchschnitt (%.1f)",
		label, currentVal, math.Abs(zScore), direction, mean)

	return &model.VMAnomaly{
		NodeID:     nodeID,
		VMID:       vm.VMID,
		VMName:     vm.Name,
		Metric:     metricName,
		Value:      currentVal,
		Mean:       mean,
		StdDev:     stddev,
		ZScore:     zScore,
		Severity:   severity,
		Message:    message,
		DetectedAt: now,
	}
}
