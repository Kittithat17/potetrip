-- name: CreateDevice :one
INSERT INTO devices (trip_id, token_hash, status, member_id, requested_name, requested_member_id, promptpay)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetDeviceByToken :one
SELECT * FROM devices WHERE token_hash = $1;

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
