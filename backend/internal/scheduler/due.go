package scheduler

import (
	"time"

	"github.com/antigravity/prometheus/internal/service/backup"
)

// isDue reports whether a cron-scheduled task should run at `now`, given its
// last run time (nil if it has never run) and its creation time, which is used
// as the base when there is no prior run. An invalid cron expression is treated
// as never-due so a misconfigured rule cannot crash the scheduler.
func isDue(cron string, last *time.Time, created, now time.Time) bool {
	base := created
	if last != nil {
		base = *last
	}
	next, err := backup.NextRun(cron, base)
	if err != nil {
		return false
	}
	return !next.After(now)
}
