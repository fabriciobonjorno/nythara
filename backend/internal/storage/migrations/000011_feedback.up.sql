-- Recado do jogador no Alpha. É opcional por definição: nada no produto
-- depende desta tabela estar preenchida.
CREATE TABLE IF NOT EXISTS feedback (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    match_id uuid REFERENCES matches(id) ON DELETE SET NULL,
    ruleset_version text NOT NULL,
    message text NOT NULL CHECK (char_length(message) BETWEEN 1 AND 2000),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS feedback_created_at_idx ON feedback (created_at DESC);
-- Um recado por partida: o convite aparece uma vez, não vira formulário.
CREATE UNIQUE INDEX IF NOT EXISTS feedback_user_match_idx
    ON feedback (user_id, match_id) WHERE match_id IS NOT NULL;
