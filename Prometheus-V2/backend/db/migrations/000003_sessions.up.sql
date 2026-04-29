SET search_path TO prom_v2;

CREATE TABLE IF NOT EXISTS sessions (
    id                  uuid PRIMARY KEY,
    user_id             uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  text NOT NULL UNIQUE,
    user_agent          text,
    ip_address          inet,
    expires_at          timestamptz NOT NULL,
    revoked_at          timestamptz,
    last_seen_at        timestamptz NOT NULL DEFAULT now(),
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sessions_user_active_idx ON sessions (user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS sessions_expires_idx ON sessions (expires_at);
