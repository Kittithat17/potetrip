-- name: UpsertUser :one
INSERT INTO users (google_sub, email, name, picture)
VALUES ($1, $2, $3, $4)
ON CONFLICT (google_sub) DO UPDATE
SET email = EXCLUDED.email, name = EXCLUDED.name, picture = EXCLUDED.picture, last_login_at = now()
RETURNING *;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3);

-- name: GetSessionUser :one
SELECT sqlc.embed(u), s.expires_at AS session_expires_at
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: ExtendSession :exec
UPDATE sessions SET expires_at = $2 WHERE token_hash = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= now();

-- name: ListUserTrips :many
-- Trips the user belongs to or has asked to join, newest first.
SELECT t.id, t.name, t.invite_code, t.base_currency, d.status, m.name AS member_name,
       COALESCE(t.owner_member_id = d.member_id, false)::bool AS is_owner
FROM devices d
JOIN trips t ON t.id = d.trip_id
LEFT JOIN members m ON m.id = d.member_id
WHERE d.user_id = $1 AND d.status <> 'rejected'
ORDER BY d.created_at DESC;
