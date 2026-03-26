package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/repository"
)

// LogRetentionJob deletes old log anomalies, analyses, network scans, and
// network anomalies to keep the database size under control.
type LogRetentionJob struct {
	logAnomalyRepo     repository.LogAnomalyRepository
	logAnalysisRepo    repository.LogAnalysisRepository
	networkScanRepo    repository.NetworkScanRepository
	networkAnomalyRepo repository.NetworkAnomalyRepository
	interval           time.Duration
}

// NewLogRetentionJob creates a new LogRetentionJob.
func NewLogRetentionJob(
	logAnomalyRepo repository.LogAnomalyRepository,
	logAnalysisRepo repository.LogAnalysisRepository,
	networkScanRepo repository.NetworkScanRepository,
	networkAnomalyRepo repository.NetworkAnomalyRepository,
	interval time.Duration,
) *LogRetentionJob {
	return &LogRetentionJob{
		logAnomalyRepo:     logAnomalyRepo,
		logAnalysisRepo:    logAnalysisRepo,
		networkScanRepo:    networkScanRepo,
		networkAnomalyRepo: networkAnomalyRepo,
		interval:           interval,
	}
}

func (j *LogRetentionJob) Name() string {
	return "log_retention"
}

func (j *LogRetentionJob) Interval() time.Duration {
	return j.interval
}

func (j *LogRetentionJob) Run(ctx context.Context) error {
	now := time.Now()
	var errs []error

	// Log anomalies: keep 90 days
	if n, err := j.logAnomalyRepo.DeleteOlderThan(ctx, now.AddDate(0, 0, -90)); err != nil {
		slog.Error("log_retention: delete old log anomalies", slog.Any("error", err))
		errs = append(errs, fmt.Errorf("delete old log anomalies: %w", err))
	} else if n > 0 {
		slog.Info("log_retention: deleted old log anomalies", slog.Int64("count", n))
	}

	// Log analyses: keep 180 days
	if n, err := j.logAnalysisRepo.DeleteOlderThan(ctx, now.AddDate(0, 0, -180)); err != nil {
		slog.Error("log_retention: delete old log analyses", slog.Any("error", err))
		errs = append(errs, fmt.Errorf("delete old log analyses: %w", err))
	} else if n > 0 {
		slog.Info("log_retention: deleted old log analyses", slog.Int64("count", n))
	}

	// Network scans: keep 30 days
	if n, err := j.networkScanRepo.DeleteOlderThan(ctx, now.AddDate(0, 0, -30)); err != nil {
		slog.Error("log_retention: delete old network scans", slog.Any("error", err))
		errs = append(errs, fmt.Errorf("delete old network scans: %w", err))
	} else if n > 0 {
		slog.Info("log_retention: deleted old network scans", slog.Int64("count", n))
	}

	// Network anomalies: keep 90 days
	if n, err := j.networkAnomalyRepo.DeleteOlderThan(ctx, now.AddDate(0, 0, -90)); err != nil {
		slog.Error("log_retention: delete old network anomalies", slog.Any("error", err))
		errs = append(errs, fmt.Errorf("delete old network anomalies: %w", err))
	} else if n > 0 {
		slog.Info("log_retention: deleted old network anomalies", slog.Int64("count", n))
	}

	if len(errs) > 0 {
		return fmt.Errorf("log retention: %d errors: %v", len(errs), errs[0])
	}
	return nil
}
