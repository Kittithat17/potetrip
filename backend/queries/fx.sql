-- name: GetFxRate :one
SELECT * FROM fx_rates
WHERE base = $1 AND quote = $2 AND requested_date = $3;

-- name: UpsertFxRate :exec
INSERT INTO fx_rates (base, quote, requested_date, rate_date, rate, fetched_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (base, quote, requested_date)
DO UPDATE SET rate_date = EXCLUDED.rate_date, rate = EXCLUDED.rate, fetched_at = now();
