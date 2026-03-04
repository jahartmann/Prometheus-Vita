CREATE TABLE metrics_history (
    id BIGSERIAL PRIMARY KEY,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cpu_usage DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_used BIGINT NOT NULL DEFAULT 0,
    memory_total BIGINT NOT NULL DEFAULT 0,
    disk_used BIGINT NOT NULL DEFAULT 0,
    disk_total BIGINT NOT NULL DEFAULT 0,
    net_in BIGINT NOT NULL DEFAULT 0,
    net_out BIGINT NOT NULL DEFAULT 0,
    load_avg DOUBLE PRECISION[] DEFAULT '{}'
);

CREATE INDEX idx_metrics_history_node_time ON metrics_history(node_id, recorded_at DESC);
