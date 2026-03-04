package scheduler

import (
	"context"
	"time"

	"github.com/antigravity/prometheus/internal/service/notification"
)

type AlertEvaluationJob struct {
	alertSvc *notification.AlertService
	interval time.Duration
}

func NewAlertEvaluationJob(alertSvc *notification.AlertService, interval time.Duration) *AlertEvaluationJob {
	return &AlertEvaluationJob{
		alertSvc: alertSvc,
		interval: interval,
	}
}

func (j *AlertEvaluationJob) Name() string {
	return "alert_evaluation"
}

func (j *AlertEvaluationJob) Interval() time.Duration {
	return j.interval
}

func (j *AlertEvaluationJob) Run(ctx context.Context) error {
	return j.alertSvc.EvaluateRules(ctx)
}
