-- +goose Up
-- Passkeys (WebAuthn) as a second way to sign in, one that works without
-- Google (e.g. in mainland China). An account may have only passkeys, so it
-- may have no Google ID or email.
ALTER TABLE users ALTER COLUMN google_sub DROP NOT NULL;
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;

-- One row per passkey. `credential` is the public key and its counters as
-- go-webauthn stores them; nothing secret lives on the server.
CREATE TABLE passkeys (
    id           bytea PRIMARY KEY, -- WebAuthn credential ID
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential   jsonb NOT NULL,
    name         text NOT NULL,     -- "iPhone", "Mac", ... for the settings list
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz NULL
);
CREATE INDEX passkeys_user_id_idx ON passkeys(user_id);

-- A WebAuthn ceremony in progress: the challenge the browser must sign.
-- Single use; the cookie holds a random token, we keep its SHA-256.
CREATE TABLE auth_challenges (
    token_hash bytea PRIMARY KEY,
    kind       text NOT NULL CHECK (kind IN ('signup', 'register', 'login')),
    user_id    uuid NULL REFERENCES users(id) ON DELETE CASCADE, -- set for 'register'
    name       text NULL,                                         -- set for 'signup'
    session    jsonb NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX auth_challenges_expires_at_idx ON auth_challenges(expires_at);

-- +goose Down
DROP TABLE auth_challenges;
DROP TABLE passkeys;
DELETE FROM users WHERE google_sub IS NULL;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
ALTER TABLE users ALTER COLUMN google_sub SET NOT NULL;
