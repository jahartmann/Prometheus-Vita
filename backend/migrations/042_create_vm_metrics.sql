CREATE TABLE IF NOT EXISTS vm_metrics_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    vmid INTEGER NOT NULL,
    vm_type VARCHAR(10) NOT NULL DEFAULT 'qemu',
    cpu_usage DOUBLE PRECISION NOT NULL DEFAULT 0,
    mem_used BIGINT NOT NULL DEFAULT 0,
    mem_total BIGINT NOT NULL DEFAULT 0,
    net_in BIGINT NOT NULL DEFAULT 0,
    net_out BIGINT NOT NULL DEFAULT 0,
    disk_read BIGINT NOT NULL DEFAULT 0,
    disk_write BIGINT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vm_metrics_node_vmid ON vm_metrics_history(node_id, vmid, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_vm_metrics_recorded_at ON vm_metrics_history(recorded_at);
