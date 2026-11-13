-- name: CreateActivity :exec
INSERT INTO trip_activity (trip_id, member_id, action, expense_id, title, amount_base)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListActivity :many
SELECT a.id, a.action, a.expense_id, a.title, a.amount_base, a.created_at, a.member_id,
       m.name AS member_name
FROM trip_activity a
LEFT JOIN members m ON m.id = a.member_id
WHERE a.trip_id = $1
ORDER BY a.created_at DESC, a.id
LIMIT 100;
