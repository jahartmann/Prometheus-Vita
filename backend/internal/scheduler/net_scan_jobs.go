package scheduler

import (
	"context"
	"time"

	"github.com/antigravity/prometheus/internal/service/netscan"
)

// NetQuickScanJob runs quick (ss-based) network scans on all nodes.
type NetQuickScanJob struct {
	scanner  *netscan.ScanScheduler
	interval time.Duration
}

// NewNetQuickScanJob creates a new NetQuickScanJob.
func NewNetQuickScanJob(scanner *netscan.ScanScheduler, interval time.Duration) *NetQuickScanJob {
	return &NetQuickScanJob{
		scanner:  scanner,
		interval: interval,
	}
}

func (j *NetQuickScanJob) Name() string {
	return "net_quick_scan"
}

func (j *NetQuickScanJob) Interval() time.Duration {
	return j.interval
}

func (j *NetQuickScanJob) Run(ctx context.Context) error {
	return j.scanner.RunQuickScans(ctx)
}

// NetFullScanJob runs full (nmap-based) network scans on all nodes.
type NetFullScanJob struct {
	scanner  *netscan.ScanScheduler
	interval time.Duration
}

// NewNetFullScanJob creates a new NetFullScanJob.
func NewNetFullScanJob(scanner *netscan.ScanScheduler, interval time.Duration) *NetFullScanJob {
	return &NetFullScanJob{
		scanner:  scanner,
		interval: interval,
	}
}

func (j *NetFullScanJob) Name() string {
	return "net_full_scan"
}

func (j *NetFullScanJob) Interval() time.Duration {
	return j.interval
}

func (j *NetFullScanJob) Run(ctx context.Context) error {
	return j.scanner.RunFullScans(ctx)
}
