-- name: CreateMember :one
INSERT INTO members (trip_id, name, promptpay)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMembers :many
SELECT * FROM members
WHERE trip_id = $1
ORDER BY created_at, id;

-- name: GetMember :one
SELECT * FROM members WHERE id = $1;

-- name: UpdateMemberPromptPay :one
UPDATE members SET promptpay = $3
WHERE id = $1 AND trip_id = $2
RETURNING *;
