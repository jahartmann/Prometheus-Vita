CREATE TABLE IF NOT EXISTS api_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    api_token_id UUID REFERENCES api_tokens(id) ON DELETE SET NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    status_code INT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    request_body JSONB,
    duration_ms INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_audit_log_user_id ON api_audit_log(user_id);
CREATE INDEX idx_api_audit_log_created_at ON api_audit_log(created_at DESC);
CREATE INDEX idx_api_audit_log_api_token_id ON api_audit_log(api_token_id);
