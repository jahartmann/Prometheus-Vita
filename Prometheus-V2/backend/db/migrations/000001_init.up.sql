CREATE SCHEMA IF NOT EXISTS prom_v2;
SET search_path TO prom_v2;

CREATE TABLE IF NOT EXISTS _v2_meta (
    key   text PRIMARY KEY,
    value text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO _v2_meta (key, value) VALUES
    ('schema_version', '1'),
    ('installed_at',   now()::text)
ON CONFLICT (key) DO NOTHING;
