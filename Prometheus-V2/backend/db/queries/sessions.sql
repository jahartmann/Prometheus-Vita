-- name: CreateSession :one
INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at;

-- name: GetSessionByRefreshHash :one
SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, last_seen_at, created_at
FROM sessions
WHERE refresh_token_hash = $1;

-- name: TouchSession :exec
UPDATE sessions
SET last_seen_at = now()
WHERE id = $1;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = now()
WHERE id = $1;

-- name: RevokeUserSessions :exec
UPDATE sessions
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < now() OR (revoked_at IS NOT NULL AND revoked_at < now() - interval '7 days');
