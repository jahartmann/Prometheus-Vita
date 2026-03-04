CREATE TABLE IF NOT EXISTS update_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    total_updates INT NOT NULL DEFAULT 0,
    security_updates INT NOT NULL DEFAULT 0,
    packages JSONB,
    error_message TEXT,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_update_checks_node_id ON update_checks(node_id);
CREATE INDEX idx_update_checks_checked_at ON update_checks(checked_at DESC);
