-- name: GetUserByID :one
SELECT id, email, name, password_hash, role, enabled, version, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, role, enabled, version, created_at, updated_at
FROM users WHERE email = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (id, email, name, password_hash, role, enabled)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, email, name, password_hash, role, enabled, version, created_at, updated_at;

-- name: UpdateUserPasswordHash :exec
UPDATE users
SET password_hash = $2,
    version = version + 1,
    updated_at = now()
WHERE id = $1;
