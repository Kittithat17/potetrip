-- name: CreatePasskeyUser :one
INSERT INTO users (id, name) VALUES ($1, $2)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: TouchUserLogin :exec
UPDATE users SET last_login_at = now() WHERE id = $1;

-- name: ListPasskeys :many
SELECT * FROM passkeys WHERE user_id = $1 ORDER BY created_at;

-- name: CreatePasskey :exec
INSERT INTO passkeys (id, user_id, credential, name) VALUES ($1, $2, $3, $4);

-- name: UpdatePasskeyUse :exec
UPDATE passkeys SET credential = $2, last_used_at = now() WHERE id = $1;

-- name: DeletePasskey :execrows
DELETE FROM passkeys WHERE id = $1 AND user_id = $2;

-- name: CountPasskeys :one
SELECT count(*) FROM passkeys WHERE user_id = $1;

-- name: CreateChallenge :exec
INSERT INTO auth_challenges (token_hash, kind, user_id, name, session, expires_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: TakeChallenge :one
-- Single use: reading a challenge deletes it.
DELETE FROM auth_challenges
WHERE token_hash = $1 AND kind = $2
RETURNING *;

-- name: DeleteExpiredChallenges :exec
DELETE FROM auth_challenges WHERE expires_at <= now();
