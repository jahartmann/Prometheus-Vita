CREATE TABLE maintenance_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    metric VARCHAR(50) NOT NULL,
    current_value DOUBLE PRECISION NOT NULL,
    predicted_value DOUBLE PRECISION NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    days_until_threshold DOUBLE PRECISION,
    slope DOUBLE PRECISION NOT NULL,
    intercept DOUBLE PRECISION NOT NULL,
    r_squared DOUBLE PRECISION NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    predicted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (node_id, metric)
);
CREATE INDEX idx_maintenance_predictions_node_id ON maintenance_predictions(node_id);
CREATE INDEX idx_maintenance_predictions_severity ON maintenance_predictions(severity);
