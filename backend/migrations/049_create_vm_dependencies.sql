CREATE TABLE IF NOT EXISTS vm_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_node_id UUID NOT NULL REFERENCES nodes(id),
    source_vmid INTEGER NOT NULL,
    target_node_id UUID NOT NULL REFERENCES nodes(id),
    target_vmid INTEGER NOT NULL,
    dependency_type VARCHAR(50) DEFAULT 'depends_on',
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
