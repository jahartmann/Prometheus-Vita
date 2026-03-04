CREATE TABLE config_backup_files (
    id UUID PRIMARY KEY,
    backup_id UUID NOT NULL REFERENCES config_backups(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    file_hash VARCHAR(64) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    file_permissions VARCHAR(10),
    file_owner VARCHAR(255),
    content BYTEA,
    diff_from_previous TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_files_backup_id ON config_backup_files(backup_id);
