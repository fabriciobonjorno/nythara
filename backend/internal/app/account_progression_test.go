package app

import (
	"context"
	"errors"
	"testing"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/progression"
)

type accountCollectionStore struct {
	domain.Store
	xp         int
	collection domain.Collection
	deck       domain.Deck
}

func (s *accountCollectionStore) AccountXP(context.Context, string) (int, error) {
	return s.xp, nil
}

func (s *accountCollectionStore) Collection(context.Context, string, string) (domain.Collection, error) {
	return s.collection, nil
}

func (s *accountCollectionStore) Deck(context.Context, string, string) (domain.Deck, error) {
	return s.deck, nil
}

func TestCollectionHidesLegendaryUntilRequiredAccountLevel(t *testing.T) {
	store := &accountCollectionStore{collection: domain.Collection{Cards: []domain.CollectionCard{
		{CardID: "VR-001", Quantity: 2}, {CardID: "VR-012", Quantity: 1},
	}}}
	service := New(store)
	principal := domain.Principal{UserID: "player"}

	locked, err := service.Collection(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(locked.Cards) != 1 || locked.Cards[0].CardID != "VR-001" {
		t.Fatalf("nível 1 expôs Lendária: %+v", locked.Cards)
	}

	store.xp = progression.MaxAccountXP
	unlocked, err := service.Collection(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(unlocked.Cards) != 2 {
		t.Fatalf("nível máximo não liberou coleção: %+v", unlocked.Cards)
	}
}

func TestBattleDeckRejectsPreviouslySavedLockedLegendary(t *testing.T) {
	store := &accountCollectionStore{
		collection: domain.Collection{Cards: []domain.CollectionCard{{CardID: "VR-012", Quantity: 1}}},
		deck:       domain.Deck{Cards: []domain.DeckCard{{CardID: "VR-012", Quantity: 1}}},
	}
	_, _, err := New(store).BattleDeck(context.Background(), domain.Principal{UserID: "player"}, "deck")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("baralho antigo com Lendária bloqueada deveria falhar: %v", err)
	}
}
