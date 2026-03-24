CREATE TABLE IF NOT EXISTS password_policy (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    min_length INTEGER NOT NULL DEFAULT 8,
    require_uppercase BOOLEAN NOT NULL DEFAULT false,
    require_lowercase BOOLEAN NOT NULL DEFAULT false,
    require_digit BOOLEAN NOT NULL DEFAULT false,
    require_special BOOLEAN NOT NULL DEFAULT false,
    max_length INTEGER NOT NULL DEFAULT 128,
    disallow_username BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Insert default policy
INSERT INTO password_policy (id, min_length, require_uppercase, require_lowercase, require_digit, require_special, max_length, disallow_username)
SELECT gen_random_uuid(), 8, false, false, false, false, 128, true
WHERE NOT EXISTS (SELECT 1 FROM password_policy);
