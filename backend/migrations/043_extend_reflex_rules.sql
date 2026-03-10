-- Add time-based scheduling and AI fields to reflex_rules
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS schedule_type TEXT NOT NULL DEFAULT 'always';
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS schedule_cron TEXT DEFAULT '';
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS time_window_start TEXT DEFAULT '';
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS time_window_end TEXT DEFAULT '';
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS time_window_days INTEGER[] DEFAULT '{}';
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS ai_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS ai_severity TEXT DEFAULT '';
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS ai_recommendation TEXT DEFAULT '';
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 0;
ALTER TABLE reflex_rules ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';
