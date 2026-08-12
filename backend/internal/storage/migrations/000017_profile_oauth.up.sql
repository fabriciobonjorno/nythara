ALTER TABLE player_profiles
    ADD COLUMN avatar_id text;

CREATE TABLE oauth_identities (
    provider text NOT NULL CHECK (provider IN ('google')),
    subject text NOT NULL CHECK (length(subject) BETWEEN 1 AND 255),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject),
    UNIQUE (provider, user_id)
);

CREATE TABLE oauth_login_tickets (
    token_hash bytea PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX oauth_login_tickets_active_idx
    ON oauth_login_tickets(expires_at)
    WHERE used_at IS NULL;
