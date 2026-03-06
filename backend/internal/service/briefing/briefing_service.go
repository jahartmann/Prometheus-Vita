package briefing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type NodeServiceInterface interface {
	GetStatus(ctx context.Context, id uuid.UUID) (*proxmox.NodeStatus, error)
	GetVMs(ctx context.Context, id uuid.UUID) ([]proxmox.VMInfo, error)
}

type Service struct {
	briefingRepo   repository.BriefingRepository
	nodeRepo       repository.NodeRepository
	metricsRepo    repository.MetricsRepository
	anomalyRepo    repository.AnomalyRepository
	predictionRepo repository.PredictionRepository
	llmRegistry    *llm.Registry
	nodeSvc        NodeServiceInterface
}

func NewService(
	briefingRepo repository.BriefingRepository,
	nodeRepo repository.NodeRepository,
	metricsRepo repository.MetricsRepository,
	anomalyRepo repository.AnomalyRepository,
	predictionRepo repository.PredictionRepository,
	llmRegistry *llm.Registry,
) *Service {
	return &Service{
		briefingRepo:   briefingRepo,
		nodeRepo:       nodeRepo,
		metricsRepo:    metricsRepo,
		anomalyRepo:    anomalyRepo,
		predictionRepo: predictionRepo,
		llmRegistry:    llmRegistry,
	}
}

func (s *Service) GenerateBriefing(ctx context.Context) error {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	now := time.Now()
	since := now.Add(-24 * time.Hour)

	var onlineCount, offlineCount int
	var summaries []model.BriefingNodeSummary

	for _, node := range nodes {
		if node.IsOnline {
			onlineCount++
		} else {
			offlineCount++
		}

		records, err := s.metricsRepo.GetByNode(ctx, node.ID, since, now)
		if err != nil || len(records) == 0 {
			summaries = append(summaries, model.BriefingNodeSummary{
				NodeID:   node.ID.String(),
				NodeName: node.Name,
				IsOnline: node.IsOnline,
			})
			continue
		}

		var cpuSum, memPctSum, diskPctSum float64
		for _, r := range records {
			cpuSum += r.CPUUsage
			if r.MemTotal > 0 {
				memPctSum += float64(r.MemUsed) / float64(r.MemTotal) * 100
			}
			if r.DiskTotal > 0 {
				diskPctSum += float64(r.DiskUsed) / float64(r.DiskTotal) * 100
			}
		}
		count := float64(len(records))

		summaries = append(summaries, model.BriefingNodeSummary{
			NodeID:   node.ID.String(),
			NodeName: node.Name,
			IsOnline: node.IsOnline,
			CPUAvg:   cpuSum / count,
			MemPct:   memPctSum / count,
			DiskPct:  diskPctSum / count,
		})
	}

	anomalies, _ := s.anomalyRepo.ListUnresolved(ctx)
	predictions, _ := s.predictionRepo.ListCritical(ctx)

	briefingData := model.BriefingData{
		TotalNodes:          len(nodes),
		OnlineNodes:         onlineCount,
		OfflineNodes:        offlineCount,
		UnresolvedAnomalies: len(anomalies),
		CriticalPredictions: len(predictions),
		NodeSummaries:       summaries,
	}

	dataJSON, _ := json.Marshal(briefingData)

	// Generate summary via LLM
	summary := s.generateSummaryWithLLM(ctx, briefingData)

	briefing := &model.MorningBriefing{
		Summary: summary,
		Data:    dataJSON,
	}

	if err := s.briefingRepo.Create(ctx, briefing); err != nil {
		return fmt.Errorf("create briefing: %w", err)
	}

	slog.Info("morning briefing generated", slog.String("id", briefing.ID.String()))
	return nil
}

func (s *Service) generateSummaryWithLLM(ctx context.Context, data model.BriefingData) string {
	modelName := s.llmRegistry.DefaultModel()
	provider, err := s.llmRegistry.GetForModel(modelName)
	if err != nil {
		return s.generateFallbackSummary(data)
	}

	dataJSON, _ := json.MarshalIndent(data, "", "  ")

	prompt := fmt.Sprintf(`Du bist ein Infrastruktur-Assistent. Erstelle ein kurzes Morning Briefing auf Deutsch basierend auf folgenden Daten:

%s

Fasse den aktuellen Zustand der Infrastruktur zusammen: Wie viele Nodes sind online/offline, gibt es Anomalien oder kritische Vorhersagen? Halte es unter 200 Woertern.`, string(dataJSON))

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		Model: modelName,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		slog.Warn("briefing LLM call failed, using fallback", slog.Any("error", err))
		return s.generateFallbackSummary(data)
	}

	return resp.Content
}

func (s *Service) generateFallbackSummary(data model.BriefingData) string {
	return fmt.Sprintf("Morning Briefing: %d/%d Nodes online. %d ungeloeste Anomalien. %d kritische Vorhersagen.",
		data.OnlineNodes, data.TotalNodes, data.UnresolvedAnomalies, data.CriticalPredictions)
}

func (s *Service) GetLatest(ctx context.Context) (*model.MorningBriefing, error) {
	return s.briefingRepo.GetLatest(ctx)
}

func (s *Service) List(ctx context.Context, limit int) ([]model.MorningBriefing, error) {
	return s.briefingRepo.List(ctx, limit)
}

func (s *Service) SetNodeService(nodeSvc NodeServiceInterface) {
	s.nodeSvc = nodeSvc
}

type LiveBriefingSummary struct {
	NodesOnline      int                  `json:"nodes_online"`
	NodesOffline     int                  `json:"nodes_offline"`
	NodesTotal       int                  `json:"nodes_total"`
	VMsRunning       int                  `json:"vms_running"`
	VMsStopped       int                  `json:"vms_stopped"`
	VMsTotal         int                  `json:"vms_total"`
	AvgCPU           float64              `json:"avg_cpu"`
	AvgRAM           float64              `json:"avg_ram"`
	AvgDisk          float64              `json:"avg_disk"`
	TopNodesByCPU    []NodeCPURank        `json:"top_nodes_by_cpu"`
	TopVMsByRAM      []VMRAMRank          `json:"top_vms_by_ram"`
	Anomalies        int                  `json:"unresolved_anomalies"`
	Predictions      int                  `json:"critical_predictions"`
	NodeDetails      []LiveNodeDetail     `json:"node_details"`
}

type NodeCPURank struct {
	NodeID   string  `json:"node_id"`
	NodeName string  `json:"node_name"`
	CPUUsage float64 `json:"cpu_usage"`
}

type VMRAMRank struct {
	NodeID     string  `json:"node_id"`
	NodeName   string  `json:"node_name"`
	VMID       int     `json:"vmid"`
	VMName     string  `json:"vm_name"`
	MemUsedPct float64 `json:"mem_used_pct"`
	MemUsed    int64   `json:"mem_used"`
	MemTotal   int64   `json:"mem_total"`
}

type LiveNodeDetail struct {
	NodeID    string  `json:"node_id"`
	NodeName  string  `json:"node_name"`
	IsOnline  bool    `json:"is_online"`
	CPUUsage  float64 `json:"cpu_usage"`
	MemPct    float64 `json:"mem_pct"`
	DiskPct   float64 `json:"disk_pct"`
	VMCount   int     `json:"vm_count"`
	VMRunning int     `json:"vm_running"`
	Uptime    int64   `json:"uptime"`
}

func (s *Service) GetLiveSummary(ctx context.Context) (*LiveBriefingSummary, error) {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	summary := &LiveBriefingSummary{}
	summary.NodesTotal = len(nodes)

	var cpuValues []float64
	var ramValues []float64
	var diskValues []float64
	var allVMs []VMRAMRank

	for _, node := range nodes {
		if !node.IsOnline {
			summary.NodesOffline++
			summary.NodeDetails = append(summary.NodeDetails, LiveNodeDetail{
				NodeID:   node.ID.String(),
				NodeName: node.Name,
				IsOnline: false,
			})
			continue
		}
		summary.NodesOnline++

		detail := LiveNodeDetail{
			NodeID:   node.ID.String(),
			NodeName: node.Name,
			IsOnline: true,
		}

		if s.nodeSvc != nil {
			status, err := s.nodeSvc.GetStatus(ctx, node.ID)
			if err == nil && status != nil {
				detail.CPUUsage = status.CPUUsage
				detail.Uptime = status.Uptime
				detail.VMCount = status.VMCount + status.CTCount
				detail.VMRunning = status.VMRunning + status.CTRunning
				summary.VMsRunning += status.VMRunning + status.CTRunning
				summary.VMsTotal += status.VMCount + status.CTCount

				if status.MemTotal > 0 {
					detail.MemPct = float64(status.MemUsed) / float64(status.MemTotal) * 100
				}
				if status.DiskTotal > 0 {
					detail.DiskPct = float64(status.DiskUsed) / float64(status.DiskTotal) * 100
				}

				cpuValues = append(cpuValues, detail.CPUUsage)
				ramValues = append(ramValues, detail.MemPct)
				diskValues = append(diskValues, detail.DiskPct)
			}

			vms, err := s.nodeSvc.GetVMs(ctx, node.ID)
			if err == nil {
				for _, vm := range vms {
					if vm.Status == "running" && vm.MaxMem > 0 {
						allVMs = append(allVMs, VMRAMRank{
							NodeID:     node.ID.String(),
							NodeName:   node.Name,
							VMID:       vm.VMID,
							VMName:     vm.Name,
							MemUsedPct: float64(vm.Mem) / float64(vm.MaxMem) * 100,
							MemUsed:    vm.Mem,
							MemTotal:   vm.MaxMem,
						})
					}
					if vm.Status != "running" {
						summary.VMsStopped++
					}
				}
			}
		}

		summary.NodeDetails = append(summary.NodeDetails, detail)
	}

	// Averages
	if len(cpuValues) > 0 {
		var sum float64
		for _, v := range cpuValues {
			sum += v
		}
		summary.AvgCPU = sum / float64(len(cpuValues))
	}
	if len(ramValues) > 0 {
		var sum float64
		for _, v := range ramValues {
			sum += v
		}
		summary.AvgRAM = sum / float64(len(ramValues))
	}
	if len(diskValues) > 0 {
		var sum float64
		for _, v := range diskValues {
			sum += v
		}
		summary.AvgDisk = sum / float64(len(diskValues))
	}

	// Top 3 nodes by CPU
	sort.Slice(summary.NodeDetails, func(i, j int) bool {
		return summary.NodeDetails[i].CPUUsage > summary.NodeDetails[j].CPUUsage
	})
	for i := 0; i < len(summary.NodeDetails) && i < 3; i++ {
		d := summary.NodeDetails[i]
		if d.IsOnline {
			summary.TopNodesByCPU = append(summary.TopNodesByCPU, NodeCPURank{
				NodeID:   d.NodeID,
				NodeName: d.NodeName,
				CPUUsage: d.CPUUsage,
			})
		}
	}

	// Top 3 VMs by RAM
	sort.Slice(allVMs, func(i, j int) bool {
		return allVMs[i].MemUsedPct > allVMs[j].MemUsedPct
	})
	for i := 0; i < len(allVMs) && i < 3; i++ {
		summary.TopVMsByRAM = append(summary.TopVMsByRAM, allVMs[i])
	}

	// Anomalies & Predictions
	anomalies, _ := s.anomalyRepo.ListUnresolved(ctx)
	predictions, _ := s.predictionRepo.ListCritical(ctx)
	summary.Anomalies = len(anomalies)
	summary.Predictions = len(predictions)

	// VMsStopped is calculated from total minus running
	summary.VMsStopped = summary.VMsTotal - summary.VMsRunning

	return summary, nil
}
