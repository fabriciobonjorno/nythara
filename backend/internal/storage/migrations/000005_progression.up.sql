-- P1 — progressão e retenção: rituais diários, maestria de Campeão, rating
-- ranked por temporada e carteira de fragmentos. Todo crédito de fragmento
-- passa por economy_transactions (auditável); o progresso é derivado
-- exclusivamente dos eventos authoritative das partidas.

CREATE TABLE player_rituals (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day date NOT NULL,
    ritual_id text NOT NULL,
    progress integer NOT NULL DEFAULT 0 CHECK (progress >= 0),
    target integer NOT NULL CHECK (target > 0),
    reward integer NOT NULL CHECK (reward > 0),
    completed_at timestamptz,
    PRIMARY KEY (user_id, day, ritual_id)
);

CREATE TABLE player_champion_mastery (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    champion_id text NOT NULL,
    xp integer NOT NULL DEFAULT 0 CHECK (xp >= 0),
    games integer NOT NULL DEFAULT 0 CHECK (games >= 0),
    wins integer NOT NULL DEFAULT 0 CHECK (wins >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, champion_id)
);

CREATE TABLE ranked_ratings (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    season_id uuid NOT NULL REFERENCES seasons(id),
    rating integer NOT NULL DEFAULT 1000 CHECK (rating >= 0),
    games integer NOT NULL DEFAULT 0 CHECK (games >= 0),
    wins integer NOT NULL DEFAULT 0 CHECK (wins >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, season_id)
);

CREATE INDEX ranked_ladder ON ranked_ratings (season_id, rating DESC, updated_at);

CREATE TABLE player_wallets (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    fragments integer NOT NULL DEFAULT 0 CHECK (fragments >= 0),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Idempotência da gravação de progresso: uma partida credita no máximo uma vez.
CREATE TABLE match_progress_log (
    match_id uuid PRIMARY KEY REFERENCES matches(id) ON DELETE CASCADE,
    recorded_at timestamptz NOT NULL DEFAULT now()
);
