package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Job interface {
	Name() string
	Interval() time.Duration
	Run(ctx context.Context) error
}

type Scheduler struct {
	jobs   []Job
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func New() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) AddJob(job Job) {
	s.jobs = append(s.jobs, job)
}

func (s *Scheduler) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)

	for _, job := range s.jobs {
		s.wg.Add(1)
		go s.runJob(ctx, job)
	}

	slog.Info("scheduler started", slog.Int("jobs", len(s.jobs)))
}

func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	slog.Info("scheduler stopped")
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	defer s.wg.Done()

	slog.Info("starting job", slog.String("job", job.Name()), slog.Duration("interval", job.Interval()))

	// Run immediately on start
	s.safeRun(ctx, job)

	ticker := time.NewTicker(job.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.safeRun(ctx, job)
		}
	}
}

// safeRun executes a job with panic recovery to prevent a single job from
// crashing the entire scheduler.
func (s *Scheduler) safeRun(ctx context.Context, job Job) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("job panicked", slog.String("job", job.Name()), slog.Any("panic", r))
		}
	}()
	if err := job.Run(ctx); err != nil {
		slog.Error("job failed", slog.String("job", job.Name()), slog.Any("error", err))
	}
}
