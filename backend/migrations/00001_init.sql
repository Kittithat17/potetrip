-- +goose Up
CREATE TABLE trips (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name          text NOT NULL,
    base_currency char(3) NOT NULL DEFAULT 'THB',
    invite_code   text UNIQUE NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE members (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id    uuid NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    name       text NOT NULL,
    promptpay  text NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX members_trip_id_idx ON members(trip_id);

CREATE TABLE slips (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     uuid NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    uploaded_by uuid NOT NULL REFERENCES members(id),
    status      text NOT NULL CHECK (status IN ('pending', 'parsed', 'confirmed', 'failed')),
    raw_result  jsonb NULL,
    error       text NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX slips_trip_id_idx ON slips(trip_id);

CREATE TABLE expenses (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     uuid NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    title       text NOT NULL,
    category    text NOT NULL CHECK (category IN ('food', 'transport', 'lodging', 'shopping', 'activity', 'other')),
    paid_by     uuid NOT NULL REFERENCES members(id),
    amount      bigint NOT NULL CHECK (amount > 0),
    currency    char(3) NOT NULL,
    fx_rate     numeric(18,8) NOT NULL,
    amount_base bigint NOT NULL,
    split_type  text NOT NULL CHECK (split_type IN ('equal', 'exact', 'items')),
    spent_at    timestamptz NOT NULL,
    slip_id     uuid NULL REFERENCES slips(id) ON DELETE SET NULL,
    created_by  uuid NOT NULL REFERENCES members(id),
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX expenses_trip_spent_idx ON expenses(trip_id, spent_at DESC);

CREATE TABLE expense_shares (
    expense_id  uuid NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    member_id   uuid NOT NULL REFERENCES members(id),
    amount_base bigint NOT NULL,
    PRIMARY KEY (expense_id, member_id)
);

CREATE TABLE expense_items (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id uuid NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    name       text NOT NULL,
    amount     bigint NOT NULL,
    member_ids uuid[] NOT NULL
);
CREATE INDEX expense_items_expense_id_idx ON expense_items(expense_id);

CREATE TABLE settlements_paid (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     uuid NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    from_member uuid NOT NULL REFERENCES members(id),
    to_member   uuid NOT NULL REFERENCES members(id),
    amount_base bigint NOT NULL CHECK (amount_base > 0),
    paid_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (from_member <> to_member)
);
CREATE INDEX settlements_paid_trip_id_idx ON settlements_paid(trip_id);

-- +goose Down
DROP TABLE settlements_paid;
DROP TABLE expense_items;
DROP TABLE expense_shares;
DROP TABLE expenses;
DROP TABLE slips;
DROP TABLE members;
DROP TABLE trips;
