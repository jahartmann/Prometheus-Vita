package vm

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type RightsizingService struct {
	nodeRepo      repository.NodeRepository
	clientFactory proxmox.ClientFactory
}

func NewRightsizingService(
	nodeRepo repository.NodeRepository,
	clientFactory proxmox.ClientFactory,
) *RightsizingService {
	return &RightsizingService{
		nodeRepo:      nodeRepo,
		clientFactory: clientFactory,
	}
}

// AnalyzeVM provides per-VM rightsizing recommendations comparing 7-day avg usage vs allocated resources.
func (s *RightsizingService) AnalyzeVM(ctx context.Context, nodeID uuid.UUID, vmid int) (*model.VMRightsizingRecommendation, error) {
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

	rrdData, err := client.GetVMRRDData(ctx, pveNodes[0], vmid, vm.Type, "week")
	if err != nil || len(rrdData) == 0 {
		return &model.VMRightsizingRecommendation{
			NodeID:     nodeID,
			VMID:       vmid,
			VMName:     vm.Name,
			VMType:     vm.Type,
			Resources:  []model.VMResourceRecommendation{},
			AnalyzedAt: time.Now(),
		}, nil
	}

	var resources []model.VMResourceRecommendation

	// CPU analysis
	var totalCPU, maxCPU float64
	for _, dp := range rrdData {
		totalCPU += dp.CPU * 100
		if dp.CPU*100 > maxCPU {
			maxCPU = dp.CPU * 100
		}
	}
	avgCPU := totalCPU / float64(len(rrdData))

	cpuRec := model.VMResourceRecommendation{
		Resource:     "cpu",
		CurrentValue: fmt.Sprintf("%d vCPU", vm.CPUs),
		AvgUsage:     avgCPU,
		MaxUsage:     maxCPU,
	}

	if avgCPU < 10 && vm.CPUs > 1 {
		newCPUs := max(vm.CPUs/2, 1)
		cpuRec.RecommendedValue = fmt.Sprintf("%d vCPU", newCPUs)
		cpuRec.Status = "reduce"
		cpuRec.Reason = fmt.Sprintf("Durchschnittliche CPU-Auslastung %.1f%% - Reduktion von %d auf %d vCPU moeglich", avgCPU, vm.CPUs, newCPUs)
	} else if avgCPU > 80 {
		newCPUs := vm.CPUs * 2
		cpuRec.RecommendedValue = fmt.Sprintf("%d vCPU", newCPUs)
		cpuRec.Status = "increase"
		cpuRec.Reason = fmt.Sprintf("Durchschnittliche CPU-Auslastung %.1f%% - Erhoehung auf %d vCPU empfohlen", avgCPU, newCPUs)
	} else {
		cpuRec.RecommendedValue = cpuRec.CurrentValue
		cpuRec.Status = "optimal"
		cpuRec.Reason = "CPU-Zuweisung ist optimal"
	}
	resources = append(resources, cpuRec)

	// Memory analysis
	if vm.MaxMem > 0 {
		var totalMem, maxMem float64
		for _, dp := range rrdData {
			if dp.MaxMem > 0 {
				usage := (dp.Mem / dp.MaxMem) * 100
				totalMem += usage
				if usage > maxMem {
					maxMem = usage
				}
			}
		}
		avgMem := totalMem / float64(len(rrdData))

		memRec := model.VMResourceRecommendation{
			Resource:     "memory",
			CurrentValue: formatBytes(vm.MaxMem),
			AvgUsage:     avgMem,
			MaxUsage:     maxMem,
		}

		if avgMem < 20 && vm.MaxMem > 512*1024*1024 {
			newMem := max(vm.MaxMem/2, 512*1024*1024)
			memRec.RecommendedValue = formatBytes(newMem)
			memRec.Status = "reduce"
			memRec.Reason = fmt.Sprintf("Durchschnittliche RAM-Nutzung %.1f%% - Reduktion von %s auf %s moeglich", avgMem, formatBytes(vm.MaxMem), formatBytes(newMem))
		} else if avgMem > 85 {
			newMem := int64(float64(vm.MaxMem) * 1.5)
			memRec.RecommendedValue = formatBytes(newMem)
			memRec.Status = "increase"
			memRec.Reason = fmt.Sprintf("Durchschnittliche RAM-Nutzung %.1f%% - Erhoehung auf %s empfohlen", avgMem, formatBytes(newMem))
		} else {
			memRec.RecommendedValue = memRec.CurrentValue
			memRec.Status = "optimal"
			memRec.Reason = "RAM-Zuweisung ist optimal"
		}
		resources = append(resources, memRec)
	}

	// Disk analysis
	if vm.MaxDisk > 0 {
		diskUsage := float64(vm.Disk) / float64(vm.MaxDisk) * 100

		diskRec := model.VMResourceRecommendation{
			Resource:     "disk",
			CurrentValue: formatBytes(vm.MaxDisk),
			AvgUsage:     diskUsage,
			MaxUsage:     diskUsage,
		}

		if diskUsage > 85 {
			newDisk := int64(float64(vm.MaxDisk) * 1.5)
			diskRec.RecommendedValue = formatBytes(newDisk)
			diskRec.Status = "increase"
			diskRec.Reason = fmt.Sprintf("Disk-Nutzung %.1f%% - Erhoehung auf %s empfohlen", diskUsage, formatBytes(newDisk))
		} else {
			diskRec.RecommendedValue = diskRec.CurrentValue
			diskRec.Status = "optimal"
			diskRec.Reason = "Disk-Zuweisung ist optimal"
		}
		resources = append(resources, diskRec)
	}

	return &model.VMRightsizingRecommendation{
		NodeID:     nodeID,
		VMID:       vmid,
		VMName:     vm.Name,
		VMType:     vm.Type,
		Resources:  resources,
		AnalyzedAt: time.Now(),
	}, nil
}

func formatBytes(b int64) string {
	const (
		MB = 1024 * 1024
		GB = 1024 * MB
		TB = 1024 * GB
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%d MB", b/MB)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
