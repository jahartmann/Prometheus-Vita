CREATE TYPE migration_status AS ENUM (
    'pending', 'preparing', 'backing_up', 'transferring',
    'restoring', 'cleaning_up', 'completed', 'failed', 'cancelled'
);

CREATE TYPE migration_mode AS ENUM ('stop', 'snapshot', 'suspend');

CREATE TABLE vm_migrations (
    id UUID PRIMARY KEY,
    source_node_id UUID NOT NULL REFERENCES nodes(id),
    target_node_id UUID NOT NULL REFERENCES nodes(id),
    vmid INT NOT NULL,
    vm_name TEXT DEFAULT '',
    vm_type TEXT DEFAULT 'qemu',
    status migration_status DEFAULT 'pending',
    mode migration_mode DEFAULT 'snapshot',
    target_storage TEXT NOT NULL,
    progress INT DEFAULT 0,
    current_step TEXT DEFAULT '',
    vzdump_file_path TEXT,
    vzdump_file_size BIGINT,
    vzdump_task_upid TEXT,
    transfer_bytes_sent BIGINT DEFAULT 0,
    transfer_speed_bps BIGINT DEFAULT 0,
    new_vmid INT,
    restore_task_upid TEXT,
    cleanup_source BOOLEAN DEFAULT true,
    cleanup_target BOOLEAN DEFAULT true,
    error_message TEXT DEFAULT '',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    initiated_by UUID REFERENCES users(id),
    CONSTRAINT source_target_differ CHECK (source_node_id != target_node_id)
);

CREATE INDEX idx_vm_migrations_source ON vm_migrations(source_node_id);
CREATE INDEX idx_vm_migrations_target ON vm_migrations(target_node_id);
CREATE INDEX idx_vm_migrations_status ON vm_migrations(status);
