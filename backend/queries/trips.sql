-- name: CreateTrip :one
INSERT INTO trips (name, base_currency, invite_code)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTripByInviteCode :one
SELECT * FROM trips WHERE invite_code = $1;

-- name: GetTrip :one
SELECT * FROM trips WHERE id = $1;
