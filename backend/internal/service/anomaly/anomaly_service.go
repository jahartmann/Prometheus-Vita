package anomaly

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
)

const (
	zScoreWarning  = 2.5
	zScoreCritical = 3.5
	minDataPoints  = 10
)

type Service struct {
	anomalyRepo repository.AnomalyRepository
	metricsRepo repository.MetricsRepository
	nodeRepo    repository.NodeRepository
}

func NewService(
	anomalyRepo repository.AnomalyRepository,
	metricsRepo repository.MetricsRepository,
	nodeRepo repository.NodeRepository,
) *Service {
	return &Service{
		anomalyRepo: anomalyRepo,
		metricsRepo: metricsRepo,
		nodeRepo:    nodeRepo,
	}
}

func (s *Service) DetectAnomalies(ctx context.Context) error {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	since := now.Add(-24 * time.Hour)

	for _, node := range nodes {
		if !node.IsOnline {
			continue
		}

		records, err := s.metricsRepo.GetByNode(ctx, node.ID, since, now)
		if err != nil {
			slog.Warn("anomaly detection: failed to get metrics", slog.String("node", node.Name), slog.Any("error", err))
			continue
		}

		if len(records) < minDataPoints {
			continue
		}

		s.checkMetric(ctx, node.ID, "cpu_usage", records, func(r model.MetricsRecord) float64 { return r.CPUUsage })

		s.checkMetric(ctx, node.ID, "memory_usage", records, func(r model.MetricsRecord) float64 {
			if r.MemTotal == 0 {
				return 0
			}
			return float64(r.MemUsed) / float64(r.MemTotal) * 100
		})

		s.checkMetric(ctx, node.ID, "disk_usage", records, func(r model.MetricsRecord) float64 {
			if r.DiskTotal == 0 {
				return 0
			}
			return float64(r.DiskUsed) / float64(r.DiskTotal) * 100
		})
	}

	return nil
}

func (s *Service) checkMetric(ctx context.Context, nodeID uuid.UUID, metric string, records []model.MetricsRecord, extract func(model.MetricsRecord) float64) {
	values := make([]float64, len(records))
	for i, r := range records {
		values[i] = extract(r)
	}

	mean, stddev := meanStdDev(values)
	if stddev == 0 {
		return
	}

	latestValue := values[len(values)-1]
	zScore := (latestValue - mean) / stddev

	if math.Abs(zScore) < zScoreWarning {
		return
	}

	severity := "warning"
	if math.Abs(zScore) >= zScoreCritical {
		severity = "critical"
	}

	record := &model.AnomalyRecord{
		NodeID:   nodeID,
		Metric:   metric,
		Value:    latestValue,
		ZScore:   zScore,
		Mean:     mean,
		StdDev:   stddev,
		Severity: severity,
	}

	if err := s.anomalyRepo.Create(ctx, record); err != nil {
		slog.Warn("anomaly detection: failed to create record",
			slog.String("node", nodeID.String()),
			slog.String("metric", metric),
			slog.Any("error", err))
	} else {
		slog.Info("anomaly detected",
			slog.String("node", nodeID.String()),
			slog.String("metric", metric),
			slog.Float64("z_score", zScore),
			slog.String("severity", severity))
	}
}

func (s *Service) ListUnresolved(ctx context.Context) ([]model.AnomalyRecord, error) {
	return s.anomalyRepo.ListUnresolved(ctx)
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.AnomalyRecord, error) {
	return s.anomalyRepo.ListByNode(ctx, nodeID)
}

func (s *Service) Resolve(ctx context.Context, id uuid.UUID) error {
	return s.anomalyRepo.Resolve(ctx, id)
}

func meanStdDev(values []float64) (float64, float64) {
	n := float64(len(values))
	if n == 0 {
		return 0, 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / n

	var varianceSum float64
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}
	stddev := math.Sqrt(varianceSum / n)

	return mean, stddev
}
