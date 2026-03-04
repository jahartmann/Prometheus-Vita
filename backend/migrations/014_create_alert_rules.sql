CREATE TYPE alert_severity AS ENUM ('info', 'warning', 'critical');

CREATE TABLE alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    node_id UUID REFERENCES nodes(id) ON DELETE CASCADE,
    metric VARCHAR(50) NOT NULL,
    operator VARCHAR(5) NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 0,
    severity alert_severity NOT NULL DEFAULT 'warning',
    channel_ids UUID[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_rules_node_id ON alert_rules(node_id);
CREATE INDEX idx_alert_rules_is_active ON alert_rules(is_active);
