-- +goose Up
-- Who created the trip. Only the owner approves join requests.
ALTER TABLE trips ADD COLUMN owner_member_id uuid NULL REFERENCES members(id) ON DELETE SET NULL;

-- A device's credential for one trip. The secret token lives only on the
-- device; we keep its SHA-256. A device starts pending and acts as a member
-- only after the owner approves it.
CREATE TABLE devices (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id             uuid NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    token_hash          bytea NOT NULL UNIQUE,
    status              text NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')),
    member_id           uuid NULL REFERENCES members(id) ON DELETE CASCADE, -- set once approved
    requested_name      text NOT NULL,
    requested_member_id uuid NULL REFERENCES members(id) ON DELETE CASCADE, -- claiming an existing name
    promptpay           text NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    decided_at          timestamptz NULL,
    CHECK (status <> 'approved' OR member_id IS NOT NULL)
);
CREATE INDEX devices_trip_status_idx ON devices(trip_id, status);

-- +goose Down
DROP TABLE devices;
ALTER TABLE trips DROP COLUMN owner_member_id;
