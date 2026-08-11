DROP TRIGGER IF EXISTS confront_composition_after_cards ON deck_cards;
DROP TRIGGER IF EXISTS confront_composition_after_deck ON decks;
DROP FUNCTION IF EXISTS validate_confront_composition_from_cards();
DROP FUNCTION IF EXISTS validate_confront_composition_from_deck();
DROP FUNCTION IF EXISTS assert_confront_composition(uuid);
