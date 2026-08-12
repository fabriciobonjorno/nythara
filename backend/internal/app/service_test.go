package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

type registrationStore struct {
	domain.Store
	publicUser  *domain.User
	invitedUser *domain.User
	sessions    int
}

func (store *registrationStore) CreateUser(_ context.Context, user domain.User, _ string) (domain.User, error) {
	store.publicUser = &user
	return user, nil
}

func (store *registrationStore) CreateInvitedAdmin(_ context.Context, _ []byte, _ time.Time,
	user domain.User, _ string) (domain.User, error) {
	store.invitedUser = &user
	return user, nil
}

func (store *registrationStore) CreateSession(context.Context, domain.NewSession) error {
	store.sessions++
	return nil
}

func TestRegisterSeparatesPublicPlayerFromInvitedAdmin(t *testing.T) {
	publicStore := &registrationStore{}
	service := NewWithClock(publicStore, func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) })
	user, _, err := service.Register(context.Background(), RegisterInput{
		Email: "PLAYER@Example.test", Password: "senha-publica-forte", Username: "Jogador_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != domain.RolePlayer || publicStore.publicUser == nil || publicStore.invitedUser != nil || publicStore.sessions != 1 {
		t.Fatalf("cadastro público saiu do caminho player: user=%+v store=%+v", user, publicStore)
	}

	inviteStore := &registrationStore{}
	service = NewWithClock(inviteStore, func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) })
	user, _, err = service.Register(context.Background(), RegisterInput{
		Email: "admin@example.test", Password: "senha-convite-forte", Username: "Guardiao_1",
		AdminInvite: strings.Repeat("a", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != domain.RoleAdmin || inviteStore.invitedUser == nil || inviteStore.publicUser != nil || inviteStore.sessions != 1 {
		t.Fatalf("convite não usou o caminho administrativo: user=%+v store=%+v", user, inviteStore)
	}
}

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
