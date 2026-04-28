SET search_path TO prom_v2;

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
    id            uuid PRIMARY KEY,
    email         citext UNIQUE NOT NULL,
    name          text NOT NULL,
    password_hash text NOT NULL,
    role          text NOT NULL,
    enabled       boolean NOT NULL DEFAULT true,
    version       int NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_role_check CHECK (role IN ('viewer', 'operator', 'admin'))
);

CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);
