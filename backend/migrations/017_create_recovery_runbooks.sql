CREATE TABLE recovery_runbooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID REFERENCES nodes(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    scenario VARCHAR(100) NOT NULL,
    steps JSONB NOT NULL,
    is_template BOOLEAN NOT NULL DEFAULT false,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_recovery_runbooks_node_id ON recovery_runbooks(node_id);
CREATE INDEX idx_recovery_runbooks_scenario ON recovery_runbooks(scenario);
