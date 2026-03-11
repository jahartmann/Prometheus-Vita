CREATE TABLE IF NOT EXISTS vm_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR(10) NOT NULL CHECK (target_type IN ('vm', 'group')),
    target_id VARCHAR(50) NOT NULL,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_vm_permissions_user ON vm_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_vm_permissions_target ON vm_permissions(target_type, target_id, node_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_vm_permissions_unique ON vm_permissions(user_id, target_type, target_id, node_id);
