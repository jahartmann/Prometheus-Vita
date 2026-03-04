CREATE TABLE IF NOT EXISTS drift_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    total_files INT NOT NULL DEFAULT 0,
    changed_files INT NOT NULL DEFAULT 0,
    added_files INT NOT NULL DEFAULT 0,
    removed_files INT NOT NULL DEFAULT 0,
    details JSONB,
    error_message TEXT,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_drift_checks_node_id ON drift_checks(node_id);
CREATE INDEX idx_drift_checks_checked_at ON drift_checks(checked_at DESC);
