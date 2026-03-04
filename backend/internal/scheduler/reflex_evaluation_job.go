package scheduler

import (
	"context"
	"time"

	"github.com/antigravity/prometheus/internal/service/reflex"
)

type ReflexEvaluationJob struct {
	reflexSvc *reflex.Service
	interval  time.Duration
}

func NewReflexEvaluationJob(reflexSvc *reflex.Service, interval time.Duration) *ReflexEvaluationJob {
	return &ReflexEvaluationJob{
		reflexSvc: reflexSvc,
		interval:  interval,
	}
}

func (j *ReflexEvaluationJob) Name() string {
	return "reflex_evaluation"
}

func (j *ReflexEvaluationJob) Interval() time.Duration {
	return j.interval
}

func (j *ReflexEvaluationJob) Run(ctx context.Context) error {
	return j.reflexSvc.EvaluateRules(ctx)
}
