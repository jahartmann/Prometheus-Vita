package scheduler

import (
	"context"
	"time"

	"github.com/antigravity/prometheus/internal/service/anomaly"
)

type AnomalyDetectionJob struct {
	anomalySvc *anomaly.Service
	interval   time.Duration
}

func NewAnomalyDetectionJob(anomalySvc *anomaly.Service, interval time.Duration) *AnomalyDetectionJob {
	return &AnomalyDetectionJob{
		anomalySvc: anomalySvc,
		interval:   interval,
	}
}

func (j *AnomalyDetectionJob) Name() string {
	return "anomaly_detection"
}

func (j *AnomalyDetectionJob) Interval() time.Duration {
	return j.interval
}

func (j *AnomalyDetectionJob) Run(ctx context.Context) error {
	return j.anomalySvc.DetectAnomalies(ctx)
}
