-- Add AI analysis and baseline management fields to drift_checks
ALTER TABLE drift_checks ADD COLUMN IF NOT EXISTS ai_analysis JSONB;
ALTER TABLE drift_checks ADD COLUMN IF NOT EXISTS baseline_updated_at TIMESTAMPTZ;
