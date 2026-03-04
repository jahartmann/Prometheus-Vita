package prediction

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
	diskThreshold   = 90.0
	memoryThreshold = 90.0
)

type Service struct {
	predictionRepo repository.PredictionRepository
	metricsRepo    repository.MetricsRepository
	nodeRepo       repository.NodeRepository
}

func NewService(
	predictionRepo repository.PredictionRepository,
	metricsRepo repository.MetricsRepository,
	nodeRepo repository.NodeRepository,
) *Service {
	return &Service{
		predictionRepo: predictionRepo,
		metricsRepo:    metricsRepo,
		nodeRepo:       nodeRepo,
	}
}

func (s *Service) RunPredictions(ctx context.Context) error {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	since := now.Add(-7 * 24 * time.Hour)

	for _, node := range nodes {
		if !node.IsOnline {
			continue
		}

		records, err := s.metricsRepo.GetByNode(ctx, node.ID, since, now)
		if err != nil {
			slog.Warn("prediction: failed to get metrics", slog.String("node", node.Name), slog.Any("error", err))
			continue
		}

		if len(records) < 20 {
			continue
		}

		s.predictMetric(ctx, node.ID, "disk_usage", diskThreshold, records, func(r model.MetricsRecord) float64 {
			if r.DiskTotal == 0 {
				return 0
			}
			return float64(r.DiskUsed) / float64(r.DiskTotal) * 100
		})

		s.predictMetric(ctx, node.ID, "memory_usage", memoryThreshold, records, func(r model.MetricsRecord) float64 {
			if r.MemTotal == 0 {
				return 0
			}
			return float64(r.MemUsed) / float64(r.MemTotal) * 100
		})
	}

	return nil
}

func (s *Service) predictMetric(ctx context.Context, nodeID uuid.UUID, metric string, threshold float64, records []model.MetricsRecord, extract func(model.MetricsRecord) float64) {
	n := len(records)
	xs := make([]float64, n)
	ys := make([]float64, n)
	baseTime := records[0].RecordedAt.Unix()

	for i, r := range records {
		xs[i] = float64(r.RecordedAt.Unix()-baseTime) / 3600.0 // hours
		ys[i] = extract(r)
	}

	slope, intercept, rSquared := linearRegression(xs, ys)

	currentValue := ys[n-1]
	currentX := xs[n-1]
	// Predict 30 days ahead
	predictedValue := slope*(currentX+30*24) + intercept

	var daysUntilThreshold *float64
	if slope > 0 && currentValue < threshold {
		hoursUntil := (threshold - intercept) / slope
		hoursFromNow := hoursUntil - currentX
		if hoursFromNow > 0 {
			days := hoursFromNow / 24.0
			daysUntilThreshold = &days
		}
	}

	severity := "info"
	if daysUntilThreshold != nil {
		if *daysUntilThreshold <= 7 {
			severity = "critical"
		} else if *daysUntilThreshold <= 30 {
			severity = "warning"
		}
	}

	pred := &model.MaintenancePrediction{
		NodeID:             nodeID,
		Metric:             metric,
		CurrentValue:       currentValue,
		PredictedValue:     predictedValue,
		Threshold:          threshold,
		DaysUntilThreshold: daysUntilThreshold,
		Slope:              slope,
		Intercept:          intercept,
		RSquared:           rSquared,
		Severity:           severity,
	}

	if err := s.predictionRepo.Upsert(ctx, pred); err != nil {
		slog.Warn("prediction: failed to upsert",
			slog.String("node", nodeID.String()),
			slog.String("metric", metric),
			slog.Any("error", err))
	}
}

func (s *Service) ListCritical(ctx context.Context) ([]model.MaintenancePrediction, error) {
	return s.predictionRepo.ListCritical(ctx)
}

func (s *Service) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]model.MaintenancePrediction, error) {
	return s.predictionRepo.ListByNode(ctx, nodeID)
}

func linearRegression(xs, ys []float64) (slope, intercept, rSquared float64) {
	n := float64(len(xs))
	if n == 0 {
		return 0, 0, 0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
		sumXY += xs[i] * ys[i]
		sumX2 += xs[i] * xs[i]
		sumY2 += ys[i] * ys[i]
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0, sumY / n, 0
	}

	slope = (n*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / n

	// R-squared
	meanY := sumY / n
	var ssTot, ssRes float64
	for i := range xs {
		predicted := slope*xs[i] + intercept
		ssRes += (ys[i] - predicted) * (ys[i] - predicted)
		ssTot += (ys[i] - meanY) * (ys[i] - meanY)
	}

	if ssTot == 0 {
		rSquared = 1
	} else {
		rSquared = math.Max(0, 1-ssRes/ssTot)
	}

	return slope, intercept, rSquared
}
