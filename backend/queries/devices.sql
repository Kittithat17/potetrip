-- name: CreateDevice :one
INSERT INTO devices (trip_id, user_id, status, member_id, requested_name, requested_member_id, promptpay)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetDeviceForUser :one
SELECT * FROM devices WHERE trip_id = $1 AND user_id = $2;

-- name: DeleteRejectedDevice :exec
DELETE FROM devices WHERE trip_id = $1 AND user_id = $2 AND status = 'rejected';

-- name: ClaimDevice :one
-- Moves a pre-login device token onto the signed-in user, then forgets the token.
UPDATE devices SET user_id = $2, token_hash = NULL
WHERE token_hash = $1 AND user_id IS NULL
RETURNING *;

-- name: GetDevice :one
SELECT * FROM devices WHERE id = $1 AND trip_id = $2;

-- name: ListPendingDevices :many
SELECT * FROM devices
WHERE trip_id = $1 AND status = 'pending'
ORDER BY created_at;

-- name: CountPendingDevices :one
SELECT count(*) FROM devices WHERE trip_id = $1 AND status = 'pending';

-- name: ApproveDevice :one
UPDATE devices SET status = 'approved', member_id = $3, decided_at = now()
WHERE id = $1 AND trip_id = $2 AND status = 'pending'
RETURNING *;

-- name: RejectDevice :one
UPDATE devices SET status = 'rejected', decided_at = now()
WHERE id = $1 AND trip_id = $2 AND status = 'pending'
RETURNING *;

-- name: SetTripOwner :exec
UPDATE trips SET owner_member_id = $2 WHERE id = $1;

-- name: DeleteDeviceForUser :exec
DELETE FROM devices WHERE trip_id = $1 AND user_id = $2;

-- name: ListPendingWithAccount :many
-- Join requests with the account that made them, so the owner can tell a
-- real friend from someone who only has the invite link.
SELECT sqlc.embed(d), u.email AS account_email, u.name AS account_name,
       (u.google_sub IS NOT NULL)::bool AS account_has_google
FROM devices d
LEFT JOIN users u ON u.id = d.user_id
WHERE d.trip_id = $1 AND d.status = 'pending'
ORDER BY d.created_at;

-- name: RejectAllPending :execrows
UPDATE devices SET status = 'rejected', decided_at = now()
WHERE trip_id = $1 AND status = 'pending';

-- name: CountUserDevices :one
SELECT count(*) FROM devices WHERE user_id = $1;
