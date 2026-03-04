CREATE TABLE IF NOT EXISTS resource_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    vmid INT NOT NULL,
    vm_name TEXT NOT NULL,
    vm_type TEXT NOT NULL, -- qemu, lxc
    resource_type TEXT NOT NULL, -- cpu, memory, disk
    current_value BIGINT NOT NULL,
    recommended_value BIGINT NOT NULL,
    avg_usage DOUBLE PRECISION NOT NULL,
    max_usage DOUBLE PRECISION NOT NULL,
    recommendation_type TEXT NOT NULL, -- downsize, upsize, optimal
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resource_recommendations_node_id ON resource_recommendations(node_id);
CREATE INDEX idx_resource_recommendations_created_at ON resource_recommendations(created_at DESC);
