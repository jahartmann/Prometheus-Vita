-- name: GetMetaValue :one
SELECT value FROM _v2_meta WHERE key = $1;

-- name: SetMetaValue :exec
INSERT INTO _v2_meta (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE
  SET value = EXCLUDED.value,
      updated_at = now();
