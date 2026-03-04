package briefing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
)

type Service struct {
	briefingRepo   repository.BriefingRepository
	nodeRepo       repository.NodeRepository
	metricsRepo    repository.MetricsRepository
	anomalyRepo    repository.AnomalyRepository
	predictionRepo repository.PredictionRepository
	llmRegistry    *llm.Registry
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
