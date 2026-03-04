-- +migrate up
CREATE TABLE alert_incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_rule_id UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'triggered',
    current_step INT NOT NULL DEFAULT 0,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id),
    last_escalated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_incidents_rule_id ON alert_incidents(alert_rule_id);
CREATE INDEX idx_alert_incidents_status ON alert_incidents(status);

ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS escalation_policy_id UUID REFERENCES escalation_policies(id);
