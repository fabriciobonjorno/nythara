package app

import (
	"testing"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

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
