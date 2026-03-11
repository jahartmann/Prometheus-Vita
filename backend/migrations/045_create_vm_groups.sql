CREATE TABLE IF NOT EXISTS vm_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    tag_filter VARCHAR(100),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vm_group_members (
    group_id UUID NOT NULL REFERENCES vm_groups(id) ON DELETE CASCADE,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    vmid INTEGER NOT NULL,
    PRIMARY KEY (group_id, node_id, vmid)
);

CREATE INDEX IF NOT EXISTS idx_vm_group_members_node_vmid ON vm_group_members(node_id, vmid);
