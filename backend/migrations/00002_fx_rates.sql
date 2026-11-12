-- +goose Up
-- Cached exchange rates. requested_date is the day asked for; rate_date is the
-- day the rate actually comes from (weekends/holidays use the last working day).
CREATE TABLE fx_rates (
    base           char(3)       NOT NULL,
    quote          char(3)       NOT NULL,
    requested_date date          NOT NULL,
    rate_date      date          NOT NULL,
    rate           numeric(18,8) NOT NULL CHECK (rate > 0),
    fetched_at     timestamptz   NOT NULL DEFAULT now(),
    PRIMARY KEY (base, quote, requested_date)
);

-- +goose Down
DROP TABLE fx_rates;
