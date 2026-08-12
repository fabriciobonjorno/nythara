DROP INDEX IF EXISTS one_active_deck_per_user_ruleset;
DROP INDEX IF EXISTS one_confront_deck_per_user;
ALTER TABLE decks
    DROP COLUMN IF EXISTS system_provided,
    DROP COLUMN IF EXISTS locked_until,
    DROP COLUMN IF EXISTS active;

-- O rollback restaura a função da migração inicial.
CREATE OR REPLACE FUNCTION assert_deck_legal(target_deck uuid) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
    deck_row decks%ROWTYPE;
    total_cards integer;
BEGIN
    SELECT * INTO deck_row FROM decks WHERE id=target_deck;
    IF NOT FOUND THEN RETURN; END IF;
    SELECT COALESCE(sum(quantity),0) INTO total_cards FROM deck_cards WHERE deck_id=target_deck;
    IF total_cards <> 36 THEN
        RAISE EXCEPTION 'deck must contain exactly 36 cards' USING ERRCODE='23514';
    END IF;
END;
$$;
