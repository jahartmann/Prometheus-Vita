CREATE TABLE IF NOT EXISTS ssh_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_type TEXT NOT NULL DEFAULT 'ed25519', -- ed25519, rsa
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL, -- encrypted
    fingerprint TEXT NOT NULL,
    is_deployed BOOLEAN NOT NULL DEFAULT false,
    deployed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ssh_key_rotation_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    interval_days INT NOT NULL DEFAULT 90,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_rotated_at TIMESTAMPTZ,
    next_rotation_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ssh_keys_node_id ON ssh_keys(node_id);
CREATE INDEX idx_ssh_key_rotation_schedules_node_id ON ssh_key_rotation_schedules(node_id);
CREATE INDEX idx_ssh_key_rotation_schedules_next_rotation ON ssh_key_rotation_schedules(next_rotation_at);
