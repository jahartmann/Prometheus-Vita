CREATE TABLE IF NOT EXISTS role_permissions (
    role TEXT PRIMARY KEY CHECK (role IN ('admin', 'operator', 'viewer')),
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO role_permissions (role, permissions)
VALUES
    ('admin', '["*"]'::jsonb),
    ('operator', '[
        "nodes.read",
        "nodes.write",
        "vms.read",
        "vms.power",
        "vms.write",
        "backups.read",
        "backups.create",
        "backups.restore",
        "backups.delete",
        "logs.read",
        "logs.manage",
        "security.read",
        "security.manage",
        "audit.read",
        "agent.use",
        "agent.execute"
    ]'::jsonb),
    ('viewer', '[
        "nodes.read",
        "vms.read",
        "backups.read",
        "logs.read",
        "security.read",
        "agent.use"
    ]'::jsonb)
ON CONFLICT (role) DO NOTHING;
