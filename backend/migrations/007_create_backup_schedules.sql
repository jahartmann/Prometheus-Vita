CREATE TABLE backup_schedules (
    id UUID PRIMARY KEY,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    cron_expression VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    retention_count INTEGER NOT NULL DEFAULT 10,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_schedules_node_id ON backup_schedules(node_id);
CREATE INDEX idx_backup_schedules_next_run ON backup_schedules(next_run_at) WHERE is_active = true;
