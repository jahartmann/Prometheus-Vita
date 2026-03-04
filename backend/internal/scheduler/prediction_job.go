package scheduler

import (
	"context"
	"time"

	"github.com/antigravity/prometheus/internal/service/prediction"
)

type PredictionJob struct {
	predictionSvc *prediction.Service
	interval      time.Duration
}

func NewPredictionJob(predictionSvc *prediction.Service, interval time.Duration) *PredictionJob {
	return &PredictionJob{
		predictionSvc: predictionSvc,
		interval:      interval,
	}
}

func (j *PredictionJob) Name() string {
	return "predictive_maintenance"
}

func (j *PredictionJob) Interval() time.Duration {
	return j.interval
}

func (j *PredictionJob) Run(ctx context.Context) error {
	return j.predictionSvc.RunPredictions(ctx)
}
