-- name: CreateExpense :one
INSERT INTO expenses (
    trip_id, title, category, paid_by, amount, currency, fx_rate,
    amount_base, split_type, spent_at, slip_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: UpdateExpense :one
UPDATE expenses SET
    title       = $3,
    category    = $4,
    paid_by     = $5,
    amount      = $6,
    currency    = $7,
    fx_rate     = $8,
    amount_base = $9,
    split_type  = $10,
    spent_at    = $11
WHERE id = $1 AND trip_id = $2
RETURNING *;

-- name: GetExpense :one
SELECT * FROM expenses WHERE id = $1 AND trip_id = $2;

-- name: DeleteExpense :execrows
DELETE FROM expenses WHERE id = $1 AND trip_id = $2;

-- name: ListExpenses :many
SELECT e.* FROM expenses e
WHERE e.trip_id = sqlc.arg(trip_id)
  AND (sqlc.narg(category)::text IS NULL OR e.category = sqlc.narg(category))
  AND (
    sqlc.narg(member_id)::uuid IS NULL
    OR e.paid_by = sqlc.narg(member_id)
    OR EXISTS (
        SELECT 1 FROM expense_shares s
        WHERE s.expense_id = e.id AND s.member_id = sqlc.narg(member_id)
    )
  )
ORDER BY e.spent_at DESC, e.created_at DESC;

-- name: ListSharesForExpenses :many
SELECT * FROM expense_shares
WHERE expense_id = ANY(sqlc.arg(expense_ids)::uuid[]);

-- name: CreateShare :exec
INSERT INTO expense_shares (expense_id, member_id, amount_base)
VALUES ($1, $2, $3);

-- name: DeleteShares :exec
DELETE FROM expense_shares WHERE expense_id = $1;
