-- Ciclo reversível da conta. A identidade permanece reservada durante o soft
-- delete; o corte de dados separa a projeção atual do histórico compartilhado.
ALTER TABLE users
    ADD COLUMN deleted_at timestamptz,
    ADD COLUMN reactivated_at timestamptz,
    ADD COLUMN reactivation_reset_pending boolean NOT NULL DEFAULT false,
    ADD COLUMN data_reset_at timestamptz,
    ADD CONSTRAINT users_reactivation_pending_player_check
        CHECK (NOT reactivation_reset_pending OR (role='player' AND deleted_at IS NULL));

ALTER TABLE decks ADD COLUMN archived_at timestamptz;

-- Nomes e estado ativo são únicos somente dentro do ciclo visível. Baralhos
-- arquivados continuam com o conteúdo original para partidas históricas.
ALTER TABLE decks DROP CONSTRAINT decks_user_id_name_key;
CREATE UNIQUE INDEX unique_visible_deck_name
    ON decks(user_id,name) WHERE archived_at IS NULL;

DROP INDEX one_active_deck_per_user_ruleset;
CREATE UNIQUE INDEX one_active_deck_per_user_ruleset
    ON decks(user_id,ruleset_version)
    WHERE active AND archived_at IS NULL;

-- A regra de um baralho de Confronto ignora os baralhos de ciclos anteriores.
CREATE OR REPLACE FUNCTION assert_one_confront_deck(target_user uuid, target_version text) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
    rules_mode text;
    deck_count integer;
BEGIN
    SELECT mode INTO rules_mode FROM rulesets WHERE version=target_version;
    IF NOT FOUND OR rules_mode<>'confront' THEN
        RETURN;
    END IF;
    SELECT count(*) INTO deck_count FROM decks
     WHERE user_id=target_user AND ruleset_version=target_version AND archived_at IS NULL;
    IF deck_count>1 THEN
        RAISE EXCEPTION 'confront ruleset permits one deck per user'
            USING ERRCODE='23505';
    END IF;
END;
$$;

CREATE INDEX users_deleted_at_idx ON users(deleted_at) WHERE deleted_at IS NOT NULL;
