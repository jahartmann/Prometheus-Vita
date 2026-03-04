CREATE TABLE dr_readiness_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    overall_score INT NOT NULL DEFAULT 0,
    backup_score INT NOT NULL DEFAULT 0,
    profile_score INT NOT NULL DEFAULT 0,
    config_score INT NOT NULL DEFAULT 0,
    details JSONB,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_dr_readiness_node_id ON dr_readiness_scores(node_id);
CREATE INDEX idx_dr_readiness_calculated_at ON dr_readiness_scores(calculated_at DESC);
