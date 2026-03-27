-- 048_create_log_tables.sql

-- Log-Quellen-Konfiguration
CREATE TABLE IF NOT EXISTS log_sources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id     UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    path        TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    is_builtin  BOOLEAN NOT NULL DEFAULT FALSE,
    parser_type TEXT NOT NULL DEFAULT 'syslog',
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (node_id, path)
);

-- Log-Anomalien
CREATE TABLE IF NOT EXISTS log_anomalies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id         UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    timestamp       TIMESTAMPTZ NOT NULL,
    source          TEXT NOT NULL,
    severity        TEXT NOT NULL,
    anomaly_score   FLOAT NOT NULL,
    category        TEXT NOT NULL,
    summary         TEXT NOT NULL,
    raw_log         TEXT NOT NULL,
    is_acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_log_anomalies_node_ts ON log_anomalies (node_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_log_anomalies_score ON log_anomalies (anomaly_score DESC);
CREATE INDEX IF NOT EXISTS idx_log_anomalies_category ON log_anomalies (category, node_id);

-- Geplante Reports
CREATE TABLE IF NOT EXISTS log_report_schedules (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cron_expression      TEXT NOT NULL,
    node_ids             UUID[] NOT NULL,
    time_window_hours    INT NOT NULL DEFAULT 24,
    delivery_channel_ids UUID[],
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at          TIMESTAMPTZ,
    next_run_at          TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Log-Analysen (Multi-Node Support)
CREATE TABLE IF NOT EXISTS log_analyses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_ids    UUID[] NOT NULL,
    time_from   TIMESTAMPTZ NOT NULL,
    time_to     TIMESTAMPTZ NOT NULL,
    report_json JSONB NOT NULL,
    model_used  TEXT NOT NULL,
    schedule_id UUID REFERENCES log_report_schedules(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_log_analyses_nodes ON log_analyses USING GIN (node_ids);
CREATE INDEX IF NOT EXISTS idx_log_analyses_time ON log_analyses (created_at DESC);

-- Log-Bookmarks
CREATE TABLE IF NOT EXISTS log_bookmarks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id        UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    anomaly_id     UUID REFERENCES log_anomalies(id) ON DELETE SET NULL,
    log_entry_json JSONB NOT NULL,
    user_note      TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
