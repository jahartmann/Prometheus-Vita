CREATE TABLE anomaly_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    metric VARCHAR(50) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    z_score DOUBLE PRECISION NOT NULL,
    mean DOUBLE PRECISION NOT NULL,
    stddev DOUBLE PRECISION NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    is_resolved BOOLEAN NOT NULL DEFAULT false,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);
CREATE INDEX idx_anomaly_records_node_id ON anomaly_records(node_id);
CREATE INDEX idx_anomaly_records_is_resolved ON anomaly_records(is_resolved);
CREATE INDEX idx_anomaly_records_detected_at ON anomaly_records(detected_at);
