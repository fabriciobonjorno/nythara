CREATE TABLE matches (
    id uuid PRIMARY KEY,
    ruleset_version text NOT NULL REFERENCES rulesets(version),
    seed bigint NOT NULL CHECK (seed >= 0),
    config jsonb NOT NULL,
    status text NOT NULL CHECK (status IN ('waiting_ready', 'active', 'finished', 'cancelled')),
    winner_slot smallint CHECK (winner_slot IN (0, 1)),
    end_reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    ended_at timestamptz
);

CREATE TABLE match_players (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    slot smallint NOT NULL CHECK (slot IN (0, 1)),
    user_id uuid NOT NULL REFERENCES users(id),
    deck_id uuid NOT NULL REFERENCES decks(id),
    ready_at timestamptz,
    last_client_sequence bigint NOT NULL DEFAULT 0 CHECK (last_client_sequence >= 0),
    PRIMARY KEY (match_id, slot),
    UNIQUE (match_id, user_id)
);

CREATE INDEX match_players_user_idx ON match_players(user_id, match_id);

CREATE TABLE match_commands (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    command_index bigint NOT NULL CHECK (command_index >= 0),
    player_slot smallint NOT NULL CHECK (player_slot IN (-1, 0, 1)),
    client_sequence bigint CHECK (client_sequence > 0),
    origin text NOT NULL CHECK (origin IN ('system', 'client', 'timeout')),
    command jsonb NOT NULL,
    accepted_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, command_index)
);

CREATE UNIQUE INDEX match_command_client_sequence
ON match_commands(match_id, player_slot, client_sequence)
WHERE client_sequence IS NOT NULL;

CREATE TABLE match_events (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    event_seq bigint NOT NULL CHECK (event_seq >= 0),
    command_index bigint NOT NULL,
    event jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, event_seq),
    FOREIGN KEY (match_id, command_index) REFERENCES match_commands(match_id, command_index) ON DELETE CASCADE
);

CREATE TABLE match_snapshots (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    command_index bigint NOT NULL CHECK (command_index >= 0),
    event_seq bigint NOT NULL CHECK (event_seq >= 0),
    snapshot jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, command_index)
);

CREATE INDEX match_snapshots_latest_idx ON match_snapshots(match_id, command_index DESC);
