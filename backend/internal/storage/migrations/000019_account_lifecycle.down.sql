DROP INDEX IF EXISTS users_deleted_at_idx;

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
     WHERE user_id=target_user AND ruleset_version=target_version;
    IF deck_count>1 THEN
        RAISE EXCEPTION 'confront ruleset permits one deck per user'
            USING ERRCODE='23505';
    END IF;
END;
$$;

DROP INDEX IF EXISTS one_active_deck_per_user_ruleset;
CREATE UNIQUE INDEX one_active_deck_per_user_ruleset
    ON decks(user_id,ruleset_version) WHERE active;

DROP INDEX IF EXISTS unique_visible_deck_name;
-- Um rollback pode encontrar nomes repetidos entre ciclos; preserva as linhas
-- históricas com um sufixo estável antes de restaurar a restrição original.
UPDATE decks SET name=left(name,43)||' [arquivado '||left(id::text,8)||']'
WHERE archived_at IS NOT NULL;
ALTER TABLE decks ADD CONSTRAINT decks_user_id_name_key UNIQUE(user_id,name);
ALTER TABLE decks DROP COLUMN archived_at;

ALTER TABLE users
    DROP CONSTRAINT users_reactivation_pending_player_check,
    DROP COLUMN data_reset_at,
    DROP COLUMN reactivation_reset_pending,
    DROP COLUMN reactivated_at,
    DROP COLUMN deleted_at;
