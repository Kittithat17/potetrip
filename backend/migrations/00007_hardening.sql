-- +goose Up
-- Who recorded each repayment (any of payer, receiver or the trip owner may).
ALTER TABLE settlements_paid ADD COLUMN created_by uuid NULL REFERENCES members(id);

-- A trip's change history: who added, edited or deleted what. Kept even
-- after the expense is gone (no foreign key on expense_id on purpose).
CREATE TABLE trip_activity (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     uuid NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    member_id   uuid NULL REFERENCES members(id),
    action      text NOT NULL CHECK (action IN ('expense_created', 'expense_updated', 'expense_deleted', 'repayment_recorded')),
    expense_id  uuid NULL,
    title       text NOT NULL,
    amount_base bigint NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX trip_activity_trip_created_idx ON trip_activity(trip_id, created_at DESC);

-- +goose Down
DROP TABLE trip_activity;
ALTER TABLE settlements_paid DROP COLUMN created_by;
