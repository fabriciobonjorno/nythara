-- Um usuário possui um único baralho em qualquer ruleset de modo Confronto,
-- inclusive versões futuras; a regra não depende mais do nome alpha-0.9.0.
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

CREATE OR REPLACE FUNCTION validate_one_confront_deck() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM assert_one_confront_deck(NEW.user_id,NEW.ruleset_version);
    RETURN NEW;
END;
$$;

CREATE CONSTRAINT TRIGGER one_confront_deck_after_deck
AFTER INSERT OR UPDATE OF user_id,ruleset_version ON decks
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_one_confront_deck();
