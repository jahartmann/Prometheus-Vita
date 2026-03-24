CREATE TABLE IF NOT EXISTS vm_tags (
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    vmid INTEGER NOT NULL,
    vm_type VARCHAR(10) NOT NULL DEFAULT 'qemu',
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (node_id, vmid, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_vm_tags_tag_id ON vm_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_vm_tags_node_vmid ON vm_tags(node_id, vmid);
