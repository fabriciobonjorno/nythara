CREATE TABLE schema_migrations (
    version bigint PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id uuid PRIMARY KEY,
    email text NOT NULL UNIQUE CHECK (email = lower(email) AND length(email) BETWEEN 3 AND 320),
    password_hash text NOT NULL,
    role text NOT NULL DEFAULT 'player' CHECK (role IN ('player', 'admin')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE player_profiles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name text NOT NULL CHECK (length(display_name) BETWEEN 2 AND 32),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE rulesets (
    version text PRIMARY KEY,
    active boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX one_active_ruleset ON rulesets (active) WHERE active;

CREATE TABLE card_definitions (
    id text NOT NULL,
    ruleset_version text NOT NULL REFERENCES rulesets(version),
    name text NOT NULL,
    faction text NOT NULL,
    card_type text NOT NULL,
    rarity text NOT NULL CHECK (rarity IN ('Comum', 'Incomum', 'Rara', 'Épica', 'Lendária')),
    cost integer NOT NULL CHECK (cost BETWEEN 0 AND 10),
    eclipse_shift integer NOT NULL CHECK (eclipse_shift BETWEEN -2 AND 2),
    sigil text NOT NULL,
    rules_text text NOT NULL,
    definition jsonb NOT NULL,
    PRIMARY KEY (id, ruleset_version)
);

CREATE TABLE champions (
    id text NOT NULL,
    ruleset_version text NOT NULL REFERENCES rulesets(version),
    name text NOT NULL,
    faction text NOT NULL,
    vitality integer NOT NULL CHECK (vitality BETWEEN 25 AND 40),
    definition jsonb NOT NULL,
    PRIMARY KEY (id, ruleset_version)
);

CREATE TABLE seasons (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    ruleset_version text NOT NULL REFERENCES rulesets(version),
    starts_at timestamptz NOT NULL,
    ends_at timestamptz,
    CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE UNIQUE INDEX one_open_season ON seasons ((ends_at IS NULL)) WHERE ends_at IS NULL;

CREATE TABLE player_cards (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id text NOT NULL,
    ruleset_version text NOT NULL,
    quantity integer NOT NULL CHECK (quantity >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, card_id, ruleset_version),
    FOREIGN KEY (card_id, ruleset_version) REFERENCES card_definitions(id, ruleset_version)
);

CREATE TABLE player_champions (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    champion_id text NOT NULL,
    ruleset_version text NOT NULL,
    acquired_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, champion_id, ruleset_version),
    FOREIGN KEY (champion_id, ruleset_version) REFERENCES champions(id, ruleset_version)
);

CREATE TABLE decks (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(name) BETWEEN 1 AND 64),
    champion_id text NOT NULL,
    ruleset_version text NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, ruleset_version),
    UNIQUE (user_id, name),
    FOREIGN KEY (champion_id, ruleset_version) REFERENCES champions(id, ruleset_version)
);

CREATE TABLE deck_cards (
    deck_id uuid NOT NULL,
    card_id text NOT NULL,
    ruleset_version text NOT NULL,
    quantity integer NOT NULL CHECK (quantity BETWEEN 1 AND 2),
    PRIMARY KEY (deck_id, card_id),
    FOREIGN KEY (deck_id, ruleset_version) REFERENCES decks(id, ruleset_version) ON DELETE CASCADE,
    FOREIGN KEY (card_id, ruleset_version) REFERENCES card_definitions(id, ruleset_version)
);

CREATE TABLE auth_sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_hash bytea NOT NULL UNIQUE,
    refresh_hash bytea NOT NULL UNIQUE,
    access_expires_at timestamptz NOT NULL,
    refresh_expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    rotated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX auth_sessions_user_idx ON auth_sessions(user_id);

CREATE TABLE auth_refresh_history (
    refresh_hash bytea PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
    consumed_at timestamptz NOT NULL
);

CREATE TABLE idempotency_keys (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation text NOT NULL,
    key text NOT NULL CHECK (length(key) BETWEEN 8 AND 128),
    request_hash bytea NOT NULL,
    response_status integer,
    response_body jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, operation, key)
);

CREATE TABLE rewards (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source text NOT NULL,
    ruleset_version text NOT NULL REFERENCES rulesets(version),
    card_id text,
    champion_id text,
    quantity integer NOT NULL CHECK (quantity > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((card_id IS NOT NULL)::integer + (champion_id IS NOT NULL)::integer = 1),
    FOREIGN KEY (card_id, ruleset_version) REFERENCES card_definitions(id, ruleset_version),
    FOREIGN KEY (champion_id, ruleset_version) REFERENCES champions(id, ruleset_version)
);

CREATE TABLE economy_transactions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind text NOT NULL,
    source text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE FUNCTION assert_deck_legal(target_deck uuid) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
    deck_row decks%ROWTYPE;
    total_cards integer;
    core_cards integer;
    allied_cards integer;
    allied_factions integer;
BEGIN
    SELECT * INTO deck_row FROM decks WHERE id = target_deck;
    IF NOT FOUND THEN
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM player_champions pc
        WHERE pc.user_id = deck_row.user_id
          AND pc.champion_id = deck_row.champion_id
          AND pc.ruleset_version = deck_row.ruleset_version
    ) THEN
        RAISE EXCEPTION 'deck champion not owned' USING ERRCODE = '23514';
    END IF;

    SELECT COALESCE(sum(dc.quantity), 0),
           COALESCE(sum(dc.quantity) FILTER (
               WHERE cd.faction = ch.faction OR cd.faction = 'Errantes'
           ), 0),
           COALESCE(sum(dc.quantity) FILTER (
               WHERE cd.faction <> ch.faction AND cd.faction <> 'Errantes'
           ), 0),
           count(DISTINCT cd.faction) FILTER (
               WHERE cd.faction <> ch.faction AND cd.faction <> 'Errantes'
           )
      INTO total_cards, core_cards, allied_cards, allied_factions
      FROM deck_cards dc
      JOIN card_definitions cd ON cd.id = dc.card_id AND cd.ruleset_version = dc.ruleset_version
      JOIN champions ch ON ch.id = deck_row.champion_id AND ch.ruleset_version = deck_row.ruleset_version
     WHERE dc.deck_id = target_deck;

    IF total_cards <> 36 THEN
        RAISE EXCEPTION 'deck must contain exactly 36 cards' USING ERRCODE = '23514';
    END IF;
    IF core_cards < 24 OR allied_cards > 12 OR allied_factions > 1 THEN
        RAISE EXCEPTION 'deck faction composition is illegal' USING ERRCODE = '23514';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM deck_cards dc
          JOIN card_definitions cd ON cd.id = dc.card_id AND cd.ruleset_version = dc.ruleset_version
         WHERE dc.deck_id = target_deck
           AND ((cd.rarity = 'Lendária' AND dc.quantity > 1) OR dc.quantity > 2)
    ) THEN
        RAISE EXCEPTION 'deck copy limit exceeded' USING ERRCODE = '23514';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM deck_cards dc
          LEFT JOIN player_cards pc
            ON pc.user_id = deck_row.user_id
           AND pc.card_id = dc.card_id
           AND pc.ruleset_version = dc.ruleset_version
         WHERE dc.deck_id = target_deck
           AND COALESCE(pc.quantity, 0) < dc.quantity
    ) THEN
        RAISE EXCEPTION 'deck contains cards not owned' USING ERRCODE = '23514';
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION validate_deck_from_decks() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM assert_deck_legal(NEW.id);
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION validate_deck_from_cards() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM assert_deck_legal(COALESCE(NEW.deck_id, OLD.deck_id));
    RETURN COALESCE(NEW, OLD);
END;
$$;

CREATE CONSTRAINT TRIGGER deck_legal_after_deck
AFTER INSERT OR UPDATE ON decks
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_deck_from_decks();

CREATE CONSTRAINT TRIGGER deck_legal_after_cards
AFTER INSERT OR UPDATE OR DELETE ON deck_cards
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_deck_from_cards();
