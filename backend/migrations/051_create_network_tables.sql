-- 049_create_network_tables.sql

CREATE TABLE IF NOT EXISTS network_scans (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id      UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    scan_type    TEXT NOT NULL,
    results_json JSONB NOT NULL,
    started_at   TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_network_scans_node_ts ON network_scans (node_id, started_at DESC);

CREATE TABLE IF NOT EXISTS network_devices (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id    UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    ip         TEXT NOT NULL,
    mac        TEXT,
    hostname   TEXT,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_known   BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (node_id, ip)
);

CREATE TABLE IF NOT EXISTS network_ports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL REFERENCES network_devices(id) ON DELETE CASCADE,
    port            INT NOT NULL,
    protocol        TEXT NOT NULL,
    state           TEXT NOT NULL,
    service_name    TEXT,
    service_version TEXT,
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_id, port, protocol)
);

CREATE TABLE IF NOT EXISTS network_anomalies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id         UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    anomaly_type    TEXT NOT NULL,
    risk_score      FLOAT NOT NULL,
    details_json    JSONB NOT NULL,
    scan_id         UUID REFERENCES network_scans(id) ON DELETE SET NULL,
    is_acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_network_anomalies_node ON network_anomalies (node_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_network_anomalies_risk ON network_anomalies (risk_score DESC);

CREATE TABLE IF NOT EXISTS scan_baselines (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id        UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    label          TEXT,
    is_active      BOOLEAN NOT NULL DEFAULT FALSE,
    baseline_json  JSONB NOT NULL,
    whitelist_json JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scan_baselines_active ON scan_baselines (node_id) WHERE is_active = TRUE;
