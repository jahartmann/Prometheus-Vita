package rightsizing

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

const (
	cpuOverprovisionThreshold  = 0.10 // avg < 10% → downsize
	cpuUnderprovisionThreshold = 0.80 // avg > 80% → upsize
	memOverprovisionThreshold  = 0.20 // avg < 20% → downsize
	memUnderprovisionThreshold = 0.85 // avg > 85% → upsize
)

type Service struct {
	recRepo       repository.RecommendationRepository
	nodeRepo      repository.NodeRepository
	clientFactory proxmox.ClientFactory
}

func NewService(
	recRepo repository.RecommendationRepository,
	nodeRepo repository.NodeRepository,
	clientFactory proxmox.ClientFactory,
) *Service {
	return &Service{
		recRepo:       recRepo,
		nodeRepo:      nodeRepo,
		clientFactory: clientFactory,
	}
}

func (s *Service) AnalyzeNode(ctx context.Context, nodeID uuid.UUID) ([]model.ResourceRecommendation, error) {
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

	// Clear old recommendations
	_ = s.recRepo.DeleteByNode(ctx, nodeID)

	var recommendations []model.ResourceRecommendation

	for _, vm := range vms {
		if vm.Status != "running" {
			continue
		}

		rrdData, err := client.GetVMRRDData(ctx, pveNodes[0], vm.VMID, vm.Type)
		if err != nil {
			slog.Warn("failed to get RRD data for VM",
				slog.Int("vmid", vm.VMID),
				slog.Any("error", err),
			)
			continue
		}

		if len(rrdData) == 0 {
			continue
		}

		// Analyze CPU
		var totalCPU, maxCPU float64
		for _, dp := range rrdData {
			totalCPU += dp.CPU
			if dp.CPU > maxCPU {
				maxCPU = dp.CPU
			}
		}
		avgCPU := totalCPU / float64(len(rrdData))

		if avgCPU < cpuOverprovisionThreshold && vm.CPUs > 1 {
			rec := model.ResourceRecommendation{
				NodeID:             nodeID,
				VMID:               vm.VMID,
				VMName:             vm.Name,
				VMType:             vm.Type,
				ResourceType:       "cpu",
				CurrentValue:       int64(vm.CPUs),
				RecommendedValue:   max(int64(float64(vm.CPUs)*0.5), 1),
				AvgUsage:           avgCPU * 100,
				MaxUsage:           maxCPU * 100,
				RecommendationType: model.RecommendationDownsize,
				Reason:             fmt.Sprintf("Durchschnittliche CPU-Auslastung %.1f%% - VM ist ueberprovisioniert", avgCPU*100),
			}
			_ = s.recRepo.Create(ctx, &rec)
			recommendations = append(recommendations, rec)
		} else if avgCPU > cpuUnderprovisionThreshold {
			rec := model.ResourceRecommendation{
				NodeID:             nodeID,
				VMID:               vm.VMID,
				VMName:             vm.Name,
				VMType:             vm.Type,
				ResourceType:       "cpu",
				CurrentValue:       int64(vm.CPUs),
				RecommendedValue:   int64(float64(vm.CPUs) * 1.5),
				AvgUsage:           avgCPU * 100,
				MaxUsage:           maxCPU * 100,
				RecommendationType: model.RecommendationUpsize,
				Reason:             fmt.Sprintf("Durchschnittliche CPU-Auslastung %.1f%% - VM benoetigt mehr Ressourcen", avgCPU*100),
			}
			_ = s.recRepo.Create(ctx, &rec)
			recommendations = append(recommendations, rec)
		}

		// Analyze Memory
		if vm.MaxMem > 0 {
			var totalMem, maxMem float64
			for _, dp := range rrdData {
				if dp.MaxMem > 0 {
					usage := dp.Mem / dp.MaxMem
					totalMem += usage
					if usage > maxMem {
						maxMem = usage
					}
				}
			}
			avgMem := totalMem / float64(len(rrdData))

			if avgMem < memOverprovisionThreshold && vm.MaxMem > 512*1024*1024 {
				rec := model.ResourceRecommendation{
					NodeID:             nodeID,
					VMID:               vm.VMID,
					VMName:             vm.Name,
					VMType:             vm.Type,
					ResourceType:       "memory",
					CurrentValue:       vm.MaxMem,
					RecommendedValue:   max(int64(float64(vm.MaxMem)*0.5), 512*1024*1024),
					AvgUsage:           avgMem * 100,
					MaxUsage:           maxMem * 100,
					RecommendationType: model.RecommendationDownsize,
					Reason:             fmt.Sprintf("Durchschnittliche Speichernutzung %.1f%% - VM ist ueberprovisioniert", avgMem*100),
				}
				_ = s.recRepo.Create(ctx, &rec)
				recommendations = append(recommendations, rec)
			} else if avgMem > memUnderprovisionThreshold {
				rec := model.ResourceRecommendation{
					NodeID:             nodeID,
					VMID:               vm.VMID,
					VMName:             vm.Name,
					VMType:             vm.Type,
					ResourceType:       "memory",
					CurrentValue:       vm.MaxMem,
					RecommendedValue:   int64(float64(vm.MaxMem) * 1.5),
					AvgUsage:           avgMem * 100,
					MaxUsage:           maxMem * 100,
					RecommendationType: model.RecommendationUpsize,
					Reason:             fmt.Sprintf("Durchschnittliche Speichernutzung %.1f%% - VM benoetigt mehr RAM", avgMem*100),
				}
				_ = s.recRepo.Create(ctx, &rec)
				recommendations = append(recommendations, rec)
			}
		}
	}

	slog.Info("rightsizing analysis completed",
		slog.String("node_id", nodeID.String()),
		slog.Int("recommendations", len(recommendations)),
	)

	return recommendations, nil
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID, limit int) ([]model.ResourceRecommendation, error) {
	return s.recRepo.ListByNode(ctx, nodeID, limit)
}

func (s *Service) ListAll(ctx context.Context, limit int) ([]model.ResourceRecommendation, error) {
	return s.recRepo.ListAll(ctx, limit)
}
