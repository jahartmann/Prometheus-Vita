-- Track when a scheduled VM action last fired so the scheduler can compute
-- due-ness from the cron expression without re-running on every tick.
ALTER TABLE scheduled_actions ADD COLUMN IF NOT EXISTS last_run_at TIMESTAMPTZ;
