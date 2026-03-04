CREATE TABLE node_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cpu_model VARCHAR(255),
    cpu_cores INT,
    cpu_threads INT,
    memory_total_bytes BIGINT,
    memory_modules JSONB,
    disks JSONB,
    network_interfaces JSONB,
    pve_version VARCHAR(50),
    kernel_version VARCHAR(100),
    installed_packages JSONB,
    storage_layout JSONB,
    custom_data JSONB
);
CREATE INDEX idx_node_profiles_node_id ON node_profiles(node_id);
CREATE INDEX idx_node_profiles_collected_at ON node_profiles(collected_at DESC);
