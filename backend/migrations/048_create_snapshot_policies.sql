CREATE TABLE IF NOT EXISTS snapshot_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id),
    vmid INTEGER NOT NULL,
    vm_type VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    keep_daily INTEGER DEFAULT 5,
    keep_weekly INTEGER DEFAULT 4,
    keep_monthly INTEGER DEFAULT 0,
    schedule_cron VARCHAR(100) NOT NULL DEFAULT '0 2 * * *',
    is_active BOOLEAN DEFAULT true,
    last_run TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS scheduled_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id),
    vmid INTEGER,
    vm_type VARCHAR(10),
    action VARCHAR(50) NOT NULL,
    schedule_cron VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
