CREATE TABLE IF NOT EXISTS migration_logs (
    id BIGSERIAL PRIMARY KEY,
    migration_id UUID NOT NULL REFERENCES vm_migrations(id) ON DELETE CASCADE,
    line TEXT NOT NULL,
    level TEXT NOT NULL DEFAULT 'info',
    phase TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_migration_logs_migration_id ON migration_logs(migration_id);
