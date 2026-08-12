-- Salão operacional: conta proprietária, convites de admin e moderação.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('player', 'admin', 'owner'));

-- Antes desta migração, "admin" era o único papel administrativo. O admin
-- mais antigo vira o proprietário inicial; os demais continuam admins, sem
-- capacidade de emitir novos convites.
UPDATE users SET role='owner'
WHERE id=(SELECT id FROM users WHERE role='admin' ORDER BY created_at,id LIMIT 1);

CREATE TABLE player_bans (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason text NOT NULL CHECK (length(reason) BETWEEN 4 AND 500),
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    lifted_by uuid REFERENCES users(id),
    lifted_at timestamptz,
    CHECK ((lifted_at IS NULL) = (lifted_by IS NULL))
);

CREATE UNIQUE INDEX one_active_player_ban
    ON player_bans(user_id) WHERE lifted_at IS NULL;
CREATE INDEX player_bans_created_idx ON player_bans(created_at DESC);

CREATE TABLE admin_invites (
    id uuid PRIMARY KEY,
    email text NOT NULL CHECK (email = lower(email) AND length(email) BETWEEN 3 AND 320),
    token_hash bytea NOT NULL UNIQUE,
    created_by uuid NOT NULL REFERENCES users(id),
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    used_by uuid REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((used_at IS NULL) = (used_by IS NULL)),
    CHECK (expires_at > created_at)
);

CREATE INDEX admin_invites_created_idx ON admin_invites(created_at DESC);
