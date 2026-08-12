package app

import (
	"strings"
	"testing"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

func TestValidateUsername(t *testing.T) {
	for _, username := range []string{"Duelista", "duelista_01", "duelista-01", "A_"} {
		if err := validateUsername(username); err != nil {
			t.Errorf("nome válido %q rejeitado: %v", username, err)
		}
	}
	for _, username := range []string{"a", strings.Repeat("a", 33), "duelista 01", " duelista", "duelista.", "duelista@", "Artífice"} {
		if err := validateUsername(username); err == nil {
			t.Errorf("nome inválido %q aceito", username)
		}
	}
}

func TestValidateDeckRejectsIllegalDeck(t *testing.T) {
	err := ValidateDeck(domain.Deck{ChampionID: firstChampionID(), Cards: nil})
	if err == nil {
		t.Fatal("deck vazio foi aceito")
	}
}

func TestValidateDeckAcceptsLegalDeck(t *testing.T) {
	deck := legalDeck(t)
	if err := ValidateDeck(deck); err != nil {
		t.Fatalf("deck legal rejeitado: %v", err)
	}
}

func legalDeck(t *testing.T) domain.Deck {
	t.Helper()
	championID := firstChampionID()
	faction := engine.Champions[championID].Faction
	deck := domain.Deck{ChampionID: championID}
	total := 0
	for _, card := range engine.CardList {
		if card.Faction != faction && card.Faction != engine.NeutralFaction {
			continue
		}
		quantity := engine.MaxCopies
		if card.Rarity == engine.RarityLendaria {
			quantity = engine.MaxLegendary
		}
		if total+quantity > engine.DeckSize {
			quantity = engine.DeckSize - total
		}
		if quantity > 0 {
			deck.Cards = append(deck.Cards, domain.DeckCard{CardID: card.ID, Quantity: quantity})
			total += quantity
		}
		if total == engine.DeckSize {
			break
		}
	}
	if total != engine.DeckSize {
		t.Fatalf("catálogo não permitiu montar deck de teste: %d cartas", total)
	}
	return deck
}

func firstChampionID() string {
	return ChampionsSorted()[0].ID
}
