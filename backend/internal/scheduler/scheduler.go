package scheduler

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
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
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		slog.Info("scheduler stopped")
	case <-time.After(30 * time.Second):
		slog.Warn("scheduler stop timed out after 30s, forcing shutdown")
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	defer s.wg.Done()
	var running atomic.Bool

	slog.Info("starting job", slog.String("job", job.Name()), slog.Duration("interval", job.Interval()))

	run := func() {
		if !running.CompareAndSwap(false, true) {
			slog.Debug("job still running, skipping", slog.String("job", job.Name()))
			return
		}
		defer running.Store(false)
		s.safeRun(ctx, job)
	}

	run() // immediate first run
	ticker := time.NewTicker(job.Interval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

// safeRun executes a job with panic recovery to prevent a single job from
// crashing the entire scheduler.
func (s *Scheduler) safeRun(ctx context.Context, job Job) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("job panicked",
				slog.String("job", job.Name()),
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()

	// Add job-level timeout (5 minutes default, or 2x interval, whichever is larger)
	timeout := 5 * time.Minute
	if job.Interval()*2 > timeout {
		timeout = job.Interval() * 2
	}
	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := job.Run(jobCtx); err != nil {
		slog.Error("job failed", slog.String("job", job.Name()), slog.Any("error", err))
	}
}
