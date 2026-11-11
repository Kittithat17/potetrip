-- name: CreateSlip :one
INSERT INTO slips (trip_id, uploaded_by, status)
VALUES ($1, $2, 'pending')
RETURNING *;

-- name: FinishSlip :exec
UPDATE slips SET status = $2, raw_result = $3, error = $4
WHERE id = $1;

-- name: GetSlip :one
SELECT * FROM slips WHERE id = $1 AND trip_id = $2;

-- name: ConfirmSlip :execrows
UPDATE slips SET status = 'confirmed'
WHERE id = $1 AND trip_id = $2 AND status = 'parsed';
