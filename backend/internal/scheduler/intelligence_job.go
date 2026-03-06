package scheduler

import (
	"context"
	"time"

	"github.com/antigravity/prometheus/internal/service/intelligence"
)

type IntelligenceJob struct {
	analysisSvc *intelligence.Service
	interval    time.Duration
}

func NewIntelligenceJob(analysisSvc *intelligence.Service, interval time.Duration) *IntelligenceJob {
	return &IntelligenceJob{
		analysisSvc: analysisSvc,
		interval:    interval,
	}
}

func (j *IntelligenceJob) Name() string {
	return "intelligence_analysis"
}

func (j *IntelligenceJob) Interval() time.Duration {
	return j.interval
}

func (j *IntelligenceJob) Run(ctx context.Context) error {
	return j.analysisSvc.RunAnalysis(ctx)
}
