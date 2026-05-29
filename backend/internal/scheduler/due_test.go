package scheduler

import (
	"testing"
	"time"
)

func TestIsDue(t *testing.T) {
	const daily = "0 2 * * *" // every day at 02:00
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 10, 3, 0, 0, 0, time.UTC)

	// Never run, created long before a passed schedule time → due.
	if !isDue(daily, nil, created, now) {
		t.Fatalf("expected due when never run and schedule time has passed")
	}

	// Ran today at 02:00; next run is tomorrow → not due at 03:00 today.
	lastToday := time.Date(2026, 1, 10, 2, 0, 0, 0, time.UTC)
	if isDue(daily, &lastToday, created, now) {
		t.Fatalf("expected NOT due immediately after today's run")
	}

	// Ran yesterday at 02:00; today's 02:00 has passed → due.
	lastYesterday := time.Date(2026, 1, 9, 2, 0, 0, 0, time.UTC)
	if !isDue(daily, &lastYesterday, created, now) {
		t.Fatalf("expected due when a scheduled time elapsed since last run")
	}

	// Invalid cron must never be due (and must not crash the scheduler).
	if isDue("nonsense", nil, created, now) {
		t.Fatalf("invalid cron must not be due")
	}
}
