-- +goose Up
-- Everyone signs in with Google. A user's access to a trip is a devices row
-- tied to the user (the per-device token from earlier versions is kept only
-- until that device signs in and claims it).
CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub    text UNIQUE NOT NULL, -- Google's stable account ID ("mock:..." in dev)
    email         text NOT NULL,
    name          text NOT NULL,
    picture       text NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz NOT NULL DEFAULT now()
);

-- A signed-in browser. The cookie holds a random token; we keep its SHA-256.
CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_user_id_idx ON sessions(user_id);
CREATE INDEX sessions_expires_at_idx ON sessions(expires_at);

ALTER TABLE devices ADD COLUMN user_id uuid NULL REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE devices ALTER COLUMN token_hash DROP NOT NULL;
-- One membership (or join request) per user per trip.
CREATE UNIQUE INDEX devices_trip_user_uidx ON devices(trip_id, user_id) WHERE user_id IS NOT NULL;
CREATE INDEX devices_user_id_idx ON devices(user_id);

-- +goose Down
DROP INDEX devices_user_id_idx;
DROP INDEX devices_trip_user_uidx;
DELETE FROM devices WHERE token_hash IS NULL;
ALTER TABLE devices ALTER COLUMN token_hash SET NOT NULL;
ALTER TABLE devices DROP COLUMN user_id;
DROP TABLE sessions;
DROP TABLE users;
