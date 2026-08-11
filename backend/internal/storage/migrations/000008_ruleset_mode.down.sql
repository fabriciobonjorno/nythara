CREATE OR REPLACE FUNCTION assert_deck_legal(target_deck uuid) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
    deck_row decks%ROWTYPE;
    total_cards integer;
    core_cards integer;
    allied_cards integer;
    allied_factions integer;
BEGIN
    SELECT * INTO deck_row FROM decks WHERE id=target_deck;
    IF NOT FOUND THEN
        RETURN;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM player_champions pc
        WHERE pc.user_id=deck_row.user_id
          AND pc.champion_id=deck_row.champion_id
          AND pc.ruleset_version=deck_row.ruleset_version
    ) THEN
        RAISE EXCEPTION 'deck avatar not owned' USING ERRCODE='23514';
    END IF;
    SELECT COALESCE(sum(dc.quantity),0) INTO total_cards
      FROM deck_cards dc WHERE dc.deck_id=target_deck;
    IF deck_row.ruleset_version='alpha-0.9.0' THEN
        IF total_cards<>30 THEN
            RAISE EXCEPTION 'confront deck must contain exactly 30 cards' USING ERRCODE='23514';
        END IF;
        IF EXISTS (
            SELECT 1 FROM deck_cards dc
            JOIN card_definitions cd
              ON cd.id=dc.card_id AND cd.ruleset_version=dc.ruleset_version
            WHERE dc.deck_id=target_deck
              AND (cd.card_type NOT IN ('Assalto','Guarda','Rito')
                   OR COALESCE((cd.definition->'confront'->>'legal')::boolean,false)=false)
        ) THEN
            RAISE EXCEPTION 'deck contains card outside Confront pool' USING ERRCODE='23514';
        END IF;
    ELSE
        SELECT COALESCE(sum(dc.quantity) FILTER (
                   WHERE cd.faction=ch.faction OR cd.faction='Errantes'
               ),0),
               COALESCE(sum(dc.quantity) FILTER (
                   WHERE cd.faction<>ch.faction AND cd.faction<>'Errantes'
               ),0),
               count(DISTINCT cd.faction) FILTER (
                   WHERE cd.faction<>ch.faction AND cd.faction<>'Errantes'
               )
          INTO core_cards,allied_cards,allied_factions
          FROM deck_cards dc
          JOIN card_definitions cd
            ON cd.id=dc.card_id AND cd.ruleset_version=dc.ruleset_version
          JOIN champions ch
            ON ch.id=deck_row.champion_id AND ch.ruleset_version=deck_row.ruleset_version
         WHERE dc.deck_id=target_deck;
        IF total_cards<>36 THEN
            RAISE EXCEPTION 'legacy deck must contain exactly 36 cards' USING ERRCODE='23514';
        END IF;
        IF core_cards<24 OR allied_cards>12 OR allied_factions>1 THEN
            RAISE EXCEPTION 'legacy deck faction composition is illegal' USING ERRCODE='23514';
        END IF;
    END IF;
    IF EXISTS (
        SELECT 1 FROM deck_cards dc
        JOIN card_definitions cd
          ON cd.id=dc.card_id AND cd.ruleset_version=dc.ruleset_version
        WHERE dc.deck_id=target_deck
          AND ((cd.rarity='Lendária' AND dc.quantity>1) OR dc.quantity>2)
    ) THEN
        RAISE EXCEPTION 'deck copy limit exceeded' USING ERRCODE='23514';
    END IF;
    IF EXISTS (
        SELECT 1 FROM deck_cards dc
        LEFT JOIN player_cards pc
          ON pc.user_id=deck_row.user_id AND pc.card_id=dc.card_id
         AND pc.ruleset_version=dc.ruleset_version
        WHERE dc.deck_id=target_deck AND COALESCE(pc.quantity,0)<dc.quantity
    ) THEN
        RAISE EXCEPTION 'deck contains cards not owned' USING ERRCODE='23514';
    END IF;
END;
$$;

ALTER TABLE rulesets DROP COLUMN mode;
