-- Proteção contra baralhos sem condição real de confronto. A composição
-- deixa 10 slots livres, mas garante que Assalto, Guarda e Rito existam em
-- quantidade suficiente para todas as fases serem jogáveis.
CREATE OR REPLACE FUNCTION assert_confront_composition(target_deck uuid) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
    rules_mode text;
    assaults integer;
    guards integer;
    rites integer;
BEGIN
    SELECT r.mode INTO rules_mode
      FROM decks d JOIN rulesets r ON r.version=d.ruleset_version
     WHERE d.id=target_deck;
    IF NOT FOUND OR rules_mode<>'confront' THEN
        RETURN;
    END IF;
    SELECT COALESCE(sum(dc.quantity) FILTER (WHERE cd.card_type='Assalto'),0),
           COALESCE(sum(dc.quantity) FILTER (WHERE cd.card_type='Guarda'),0),
           COALESCE(sum(dc.quantity) FILTER (WHERE cd.card_type='Rito'),0)
      INTO assaults,guards,rites
      FROM deck_cards dc
      JOIN card_definitions cd
        ON cd.id=dc.card_id AND cd.ruleset_version=dc.ruleset_version
     WHERE dc.deck_id=target_deck;
    IF assaults<8 OR guards<8 OR rites<4 THEN
        RAISE EXCEPTION 'confront deck requires at least 8 assaults, 8 guards and 4 rites'
            USING ERRCODE='23514';
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION validate_confront_composition_from_deck() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM assert_confront_composition(NEW.id);
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION validate_confront_composition_from_cards() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM assert_confront_composition(COALESCE(NEW.deck_id,OLD.deck_id));
    RETURN COALESCE(NEW,OLD);
END;
$$;

CREATE CONSTRAINT TRIGGER confront_composition_after_deck
AFTER INSERT OR UPDATE ON decks
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_confront_composition_from_deck();

CREATE CONSTRAINT TRIGGER confront_composition_after_cards
AFTER INSERT OR UPDATE OR DELETE ON deck_cards
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_confront_composition_from_cards();
