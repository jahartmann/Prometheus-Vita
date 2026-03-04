CREATE TYPE backup_type AS ENUM ('manual', 'scheduled', 'pre_update');
CREATE TYPE backup_status AS ENUM ('pending', 'running', 'completed', 'failed');

CREATE TABLE config_backups (
    id UUID PRIMARY KEY,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    version INTEGER NOT NULL DEFAULT 1,
    backup_type backup_type NOT NULL DEFAULT 'manual',
    file_count INTEGER NOT NULL DEFAULT 0,
    total_size BIGINT NOT NULL DEFAULT 0,
    status backup_status NOT NULL DEFAULT 'pending',
    error_message TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_config_backups_node_id ON config_backups(node_id);
CREATE INDEX idx_config_backups_created_at ON config_backups(created_at DESC);
