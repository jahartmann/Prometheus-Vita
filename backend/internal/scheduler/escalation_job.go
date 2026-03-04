package scheduler

import (
	"context"
	"time"

	"github.com/antigravity/prometheus/internal/service/notification"
)

type EscalationJob struct {
	escalationSvc *notification.EscalationService
	interval      time.Duration
}

func NewEscalationJob(escalationSvc *notification.EscalationService, interval time.Duration) *EscalationJob {
	return &EscalationJob{
		escalationSvc: escalationSvc,
		interval:      interval,
	}
}

func (j *EscalationJob) Name() string {
	return "escalation_processing"
}

func (j *EscalationJob) Interval() time.Duration {
	return j.interval
}

func (j *EscalationJob) Run(ctx context.Context) error {
	return j.escalationSvc.ProcessEscalations(ctx)
}
