-- name: SumPaidByMember :many
SELECT paid_by AS member_id, sum(amount_base)::bigint AS total
FROM expenses
WHERE trip_id = $1
GROUP BY paid_by;

-- name: SumOwedByMember :many
SELECT s.member_id, sum(s.amount_base)::bigint AS total
FROM expense_shares s
JOIN expenses e ON e.id = s.expense_id
WHERE e.trip_id = $1
GROUP BY s.member_id;

-- name: ListSettlementsPaid :many
SELECT * FROM settlements_paid
WHERE trip_id = $1
ORDER BY paid_at, id;

-- name: CreateSettlementPaid :one
INSERT INTO settlements_paid (trip_id, from_member, to_member, amount_base)
VALUES ($1, $2, $3, $4)
RETURNING *;
