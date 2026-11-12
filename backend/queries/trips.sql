-- name: CreateTrip :one
INSERT INTO trips (name, base_currency, invite_code)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTripByInviteCode :one
SELECT * FROM trips WHERE invite_code = $1;

-- name: GetTrip :one
SELECT * FROM trips WHERE id = $1;

-- name: LockTrip :one
-- Serialises currency changes against new expenses.
SELECT * FROM trips WHERE id = $1 FOR UPDATE;

-- name: TripCurrencyForShare :one
SELECT base_currency FROM trips WHERE id = $1 FOR SHARE;

-- name: CountTripActivity :one
SELECT
    (SELECT count(*) FROM expenses e WHERE e.trip_id = $1)
  + (SELECT count(*) FROM settlements_paid s WHERE s.trip_id = $1) AS n;

-- name: UpdateTrip :one
UPDATE trips SET name = $2, base_currency = $3
WHERE id = $1
RETURNING *;
