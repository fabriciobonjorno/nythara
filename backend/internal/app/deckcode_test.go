package app

import (
	"strings"
	"testing"

	"veurubro/backend/internal/domain"
	"veurubro/backend/internal/engine"
)

func TestDeckCodeRoundTripDeterministic(t *testing.T) {
	deck := domain.Deck{
		ChampionID:     "CH-CI-01",
		RulesetVersion: engine.RulesetVersion,
		Cards: []domain.DeckCard{
			{CardID: "VR-050", Quantity: 2},
			{CardID: "VR-006", Quantity: 2},
			{CardID: "VR-049", Quantity: 2},
		},
	}
	code := EncodeDeckCode(deck)
	if !strings.HasPrefix(code, "VR1.") {
		t.Fatalf("código sem prefixo versionado: %q", code)
	}
	// Ordem de entrada não muda o código (cartas são ordenadas por id).
	shuffled := deck
	shuffled.Cards = []domain.DeckCard{deck.Cards[2], deck.Cards[0], deck.Cards[1]}
	if EncodeDeckCode(shuffled) != code {
		t.Fatal("o mesmo deck deveria gerar sempre o mesmo código")
	}

	champion, version, cards, err := DecodeDeckCode("  " + code + "\n")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if champion != "CH-CI-01" || version != engine.RulesetVersion {
		t.Fatalf("cabeçalho: %s %s", champion, version)
	}
	got := map[string]int{}
	for _, card := range cards {
		got[card.CardID] = card.Quantity
	}
	for _, want := range deck.Cards {
		if got[want.CardID] != want.Quantity {
			t.Fatalf("carta %s: %d; esperado %d", want.CardID, got[want.CardID], want.Quantity)
		}
	}
}

func TestDeckCodeRejectsGarbage(t *testing.T) {
	cases := []string{
		"",
		"ABC123",
		"VR1.",
		"VR1.!!!não-base64!!!",
		"VR1." + "eyJmb28iOiJiYXIifQ", // JSON válido sem campos exigidos
	}
	for _, code := range cases {
		if _, _, _, err := DecodeDeckCode(code); err == nil {
			t.Fatalf("código %q deveria ser rejeitado", code)
		}
	}
}

func TestDeckCodeRejectsBadQuantities(t *testing.T) {
	deck := domain.Deck{ChampionID: "CH-CI-01", RulesetVersion: engine.RulesetVersion,
		Cards: []domain.DeckCard{{CardID: "VR-006", Quantity: 2}}}
	code := EncodeDeckCode(deck)
	// Corrompe a quantidade trocando o payload: gera um código com qty 0.
	zero := domain.Deck{ChampionID: "CH-CI-01", RulesetVersion: engine.RulesetVersion,
		Cards: []domain.DeckCard{{CardID: "VR-006", Quantity: 0}}}
	badCode := EncodeDeckCode(zero)
	if badCode == code {
		t.Fatal("códigos deveriam diferir")
	}
	if _, _, _, err := DecodeDeckCode(badCode); err == nil {
		t.Fatal("quantidade 0 deveria ser rejeitada")
	}
}
